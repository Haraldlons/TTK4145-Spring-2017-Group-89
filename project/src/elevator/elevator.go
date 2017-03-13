package elevator

import (
	"../definitions"
	"../driver"
	// "../storage"
	"fmt"
	"time"
)

var time50ms = 50 * time.Millisecond

func ExecuteOrders(elevatorStateChanForExecuteOrders <-chan definitions.ElevatorState, orderListForExecuteOrders chan definitions.Orders, updateElevatorStateDirection chan<- int, completedCurrentOrder chan<- bool, updateElevatorDestinationChan chan<- int) {
	elevatorState := definitions.ElevatorState{}

	go func() {
		for {
			select {
			case elevatorState = <-elevatorStateChanForExecuteOrders:
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	stopCurrentOrder := make(chan bool)
	isFirstOrder := true
	for {
		select {
		case orderList := <-orderListForExecuteOrders:
			updateElevatorDestination(orderList, updateElevatorDestinationChan, elevatorState)
			if len(orderList.Orders) > 0 {
				// fmt.Println("Hopefully going to new floor: ", orderList.Orders[0].Floor, "and if-statement: ", len(orderList.Orders) > 0)
				if !isFirstOrder {
					stopCurrentOrder <- true
					// *localOrderList = definitions.Orders{[]definitions.Order{{Floor: 3, Direction: 1},{Floor: 0, Direction: -1}}}
				}
				isFirstOrder = false
				// fmt.Println(...)
				go GoToFloor(orderList.Orders[0].Floor, elevatorState, stopCurrentOrder, completedCurrentOrder, updateElevatorStateDirection)
				time.Sleep(20 * time.Millisecond)

				// // storage.SaveOrdersToFile(1, localOrderList)
				// if len(localOrderList.Orders) > 0 {

			}
			// 	// fmt.Println("localOrderList", localOrderList.Orders)
			// 	isFirstButtonPress = false
			// 	time.Sleep(20 * time.Millisecond)
			// 	*localOrderList = definitions.Orders{localOrderList.Orders[1:]}
			// 	i++
		}
	}
}

func updateElevatorDestination(orderList definitions.Orders, updateElevatorDestinationChan chan<- int, elevatorState definitions.ElevatorState) {
	if len(orderList.Orders) == 0 {
		updateElevatorDestinationChan <- definitions.IDLE
	} else {
		switch elevatorState.Direction {
		case definitions.DIR_UP:
			maxFloor := 0
			for _, orders := range orderList.Orders {
				if orders.Floor > maxFloor {
					maxFloor = orders.Floor
				}
			}
			updateElevatorDestinationChan <- maxFloor
		case definitions.DIR_DOWN:
			minFloor := definitions.N_FLOORS
			for _, orders := range orderList.Orders {
				if orders.Floor < minFloor {
					minFloor = orders.Floor
				}
			}
			updateElevatorDestinationChan <- minFloor
		default:
			updateElevatorDestinationChan <- orderList.Orders[0].Floor
		}
	}
}

// func ListenAfterElevatStateUpdatesAndSaveToFile(elevatorState *definitions.ElevatorState, updateElevatorStateDirection chan int, updateElevatorStateFloor chan int) {
// 	for {
// 		select {
// 		case tempDirection := <-updateElevatorStateDirection:
// 			elevatorState.Direction = tempDirection
// 			storage.SaveElevatorStateToFile(elevatorState)
// 			time.Sleep(10 * time.Millisecond)
// 		case tempFloor := <-updateElevatorStateFloor:
// 			elevatorState.LastFloor = tempFloor
// 			storage.SaveElevatorStateToFile(elevatorState)
// 			time.Sleep(10 * time.Millisecond)
// 		}
// 	}
// }

func CheckForElevatorFloorUpdates(updateElevatorStateFloor chan<- int, elevatorStateForFloorUpdatesChan <-chan definitions.ElevatorState) {
	elevatorState := definitions.ElevatorState{}

	go func() {
		for {
			select {
			case elevatorState = <-elevatorStateForFloorUpdatesChan:
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	for {
		lastFloor := driver.Elev_get_floor_sensor_signal()
		if lastFloor >= 0 && lastFloor < definitions.N_FLOORS && lastFloor != elevatorState.LastFloor {
			if lastFloor == 0 {
				updateElevatorStateFloor <- 0
				// elevatorState.LastFloor = 0
				fmt.Println("Last Floor: 1. Direction: ", elevatorState.Direction, "(maybe need to use * )")
				// storage.SaveElevatorStateToFile(elevatorState)
				driver.Elev_set_floor_indicator(lastFloor)
			} else if lastFloor < (definitions.N_FLOORS - 1) {
				// if elevatorState.LastFloor > lastFloor {
				// 	elevatorState.Direction = definitions.DIR_DOWN
				// } else {
				// 	elevatorState.Direction = definitions.DIR_UP
				// }
				driver.Elev_set_floor_indicator(lastFloor)
				updateElevatorStateFloor <- lastFloor
				fmt.Println("Last Floor: ", lastFloor, ". Direction: ", elevatorState.Direction)
				// storage.SaveElevatorStateToFile(elevatorState)
			} else if lastFloor == (definitions.N_FLOORS - 1) {
				// elevatorState.Direction = definitions.DIR_DOWN
				// elevatorState.LastFloor = lastFloor
				driver.Elev_set_floor_indicator(lastFloor)
				updateElevatorStateFloor <- lastFloor
				fmt.Println("Last Floor: ", definitions.N_FLOORS, ". Direction: ", elevatorState.Direction)
				// storage.SaveElevatorStateToFile(elevatorState)
			}
		}
		time.Sleep(time.Millisecond * 10)
	}
}

/*This functions should be cleaned up. I have an ide how to do it*/
// func PrintLastFloorIfChanged(elevatorState *definitions.ElevatorState) {
// }

func GoToFloor(destinationFloor int, elevatorState definitions.ElevatorState, stopCurrentOrder chan bool, completedCurrentOrder chan<- bool, updateElevatorStateDirection chan<- int) {
	// defer fmt.Println("Exeting goToFloor to floor: ", destinationFloor)
	// storage.SaveOrderToFile(destinationFloor)
	// elevatorActive = true

	fmt.Println("Going to floor: ", destinationFloor, " (0-3) ")
	direction := elevatorState.Direction
	lastFloor := elevatorState.LastFloor

	if driver.Elev_get_floor_sensor_signal() == destinationFloor {
		// storage.SaveOrderToFile(-1)
		fmt.Println("You are allready on the desired floor")
		// elevatorActive = false
		driver.Elev_set_motor_direction(driver.DIRECTION_STOP)
		completedCurrentOrder <- true
		// endProgram = true
		for {

			select {
			case <-stopCurrentOrder:
				fmt.Println("Finially got message to stop going to floor, ", destinationFloor)
				return
			case <-time.After(2000 * time.Millisecond):
				fmt.Println("Still have not got message to kill this order to floor: ", destinationFloor)

			}
		}
		return
	} else { /*You are not on the desired floor*/
		// fmt.Println("You are not on the desired floor")
		driver.Elev_set_door_open_lamp(0)
		if lastFloor == destinationFloor {
			fmt.Println("lastFloor == destinationFloor")
			if direction == 1 {
				driver.Elev_set_motor_direction(driver.DIRECTION_DOWN)
				updateElevatorStateDirection <- driver.DIRECTION_DOWN
			} else {
				driver.Elev_set_motor_direction(driver.DIRECTION_UP)
				updateElevatorStateDirection <- driver.DIRECTION_UP
			}
		} else if lastFloor < destinationFloor {
			driver.Elev_set_motor_direction(driver.DIRECTION_UP)
			updateElevatorStateDirection <- driver.DIRECTION_UP
		} else {
			driver.Elev_set_motor_direction(driver.DIRECTION_DOWN)
			updateElevatorStateDirection <- driver.DIRECTION_DOWN
		}
		for {
			select {
			case <-stopCurrentOrder:
				fmt.Println("stopCurrentOrder recieved. Stopping to floor: ", destinationFloor)
				return
			default:
				// fmt.Println("Floor: ", driver.Elev_get_floor_sensor_signal())
				// fmt.Println("Testing")
				if driver.Elev_get_floor_sensor_signal() == destinationFloor {
					// orderList <- orderList[1:]
					fmt.Println("You reached your desired floor. Walk out\n")
					updateElevatorStateDirection <- driver.DIRECTION_STOP

					time.Sleep(time.Millisecond * 150) //So the elevator stops in the middle of the sensor
					// elevatorActive = false
					// driver.Elev_set_button_lamp(1,1,1)
					// driver.Elev_set_button_lamp(0,1,1)
					driver.Elev_set_floor_indicator(destinationFloor)
					driver.Elev_set_motor_direction(driver.DIRECTION_STOP)
					// endProgram = true
					time.Sleep(time50ms * 10)
					driver.Elev_set_door_open_lamp(1)
					// storage.SaveOrderToFile(-1)
					time.Sleep(time.Millisecond * 100)
					completedCurrentOrder <- true
					for {
						select {
						case <-stopCurrentOrder:
							// fmt.Println("Finially got message to stop going to floor, ", destinationFloor)
							return
						case <-time.After(5000 * time.Millisecond):
							fmt.Println("Still have not got message to kill this order to floor: ", destinationFloor)
						}
					}
					return
				} else if driver.Elev_get_floor_sensor_signal() == 0 { /*This is just to be fail safe*/
					driver.Elev_set_motor_direction(driver.DIRECTION_UP)
				} else if driver.Elev_get_floor_sensor_signal() == 3 {
					driver.Elev_set_motor_direction(driver.DIRECTION_DOWN)
				} else {
					time.Sleep(time50ms) // 50ms
				}
			}
		}
	}
}

func setFloorIndicator() {
	sensorValue := driver.Elev_get_floor_sensor_signal()
	if sensorValue != -1 {
		driver.Elev_set_floor_indicator(sensorValue)
	}
}

// func GetState() struct {
// 	return state
// }
