package slave

import (
	"../definitions"
	"../driver"
	// "./../slave"
	// "./src/network"
	"../buttons"
	"../elevator"
	//"./src/driver"
	// "../storage"
	//"./src/master"
	//"./src/watchdog"
	// "network"
	// "storage"
	"fmt"
	"time"
	// "encoding/json"
)

var elevatorState = definitions.ElevatorState{2, 0,0}

// var orderList = definitions.orderList
func Run() {
	fmt.Println("Slave started!")
	// Initializing
	driver.Elev_init()
	driver.Elev_set_motor_direction(driver.DIRECTION_STOP) 

	// Channel Definitions
	internalButtonsPressesChan := make(chan [definitions.N_FLOORS]int)
	externalButtonsPressesChan := make(chan [definitions.N_FLOORS][2]int)


	// Make manually orderList
	// orderList := definitions.Orders{definitions}
	listOfNumbers := []int{0,1,2,1,3}
	secondListOfNumbers := []int{-1,1,1,-1,1}

	totalOrderList := definitions.Orders{[]definitions.Order{{Floor: 3, Direction: 1},{Floor: 0, Direction: -1}}}

	for i := range listOfNumbers {
		totalOrderList = definitions.Orders{append(totalOrderList.Orders,definitions.Order{Floor: listOfNumbers[i], Direction: secondListOfNumbers[i]})}
	}

	fmt.Println("printing orderList:", totalOrderList)


	// storage.LoadElevatorStateFromFile(&elevatorState)
	// storage.LoadOrderListFromFile(&orderList)
	updatedOrderList := make(chan int)


	go elevator.ExecuteOrders(&totalOrderList, &elevatorState, updatedOrderList)
	// go elevator.CheckForElevatorStateUpdates()
	// go watchdog.SendImAliveMessages()
	// go watchdog.CheckForMasterAlive()

	go buttons.Check_button_internal(internalButtonsPressesChan)
	go buttons.Check_button_external(externalButtonsPressesChan)
	// go handleInternalButtonPresses(internalButtonsPressesChan)
	// go handleExternalButtonPresses(externalButtonsPressesChan)

	go printExternalPresses(externalButtonsPressesChan)
	go printInternalPresses(internalButtonsPressesChan)
	// go checkForUpdatesFromMaster()
	// go sendUpdatesToMaster()

	
	// buttonPressesUpdates := make(chan button)
	// go checkForButtonPresses()


	// elevatorState := definitions.ElevatorState{2, 0}

	// go elevator.PrintLastFloorIfChanged(&elevatorState)
	updatedOrderList <- 1
	for {
		time.Sleep(time.Millisecond * 10000)
		// updatedOrderList <- 1
		// fmt.Println("updatedOrderList now!")
	}
	return
}

func Change_master() bool {
	return true
}

func printExternalPresses(externalButtonsChan chan [definitions.N_FLOORS][2]int) {
	for {
		select {
		case <-externalButtonsChan:
			fmt.Println("\nExternal button pressed: ", <-externalButtonsChan)
			// go findFloorAngGoTo(externalButtonsChan)
			time.Sleep(time.Millisecond * 200)

			// default:
			// fmt.Println("No button pressed")
		}
	}
}

func printInternalPresses(internalButtonsChan chan [definitions.N_FLOORS]int) {
	// stopCurrentOrder := make(chan int) // Doesn't matter which data type.
	// isFirstButtonPress := true
	for {
		select {
		case <-internalButtonsChan:
			fmt.Println("Internal button pressed: ", <-internalButtonsChan)
			// if !isFirstButtonPress {
			// 	stopCurrentOrder <- 1
			// } //Value in channel doesn't matter
			// // if(saveOrderToFile) { go findFloorAndGoTo()}
			// go findFloorAndGoTo(internalButtonsChan, <-internalButtonsChan, stopCurrentOrder)
			// time.Sleep(time.Millisecond * 200)
			// isFirstButtonPress = false
			// default:
			// fmt.Println("No button pressed")
		}
	}
}

// func findFloorAndGoTo(kanal chan [definitions.N_FLOORS]int, buttonPresses [definitions.N_FLOORS]int, stopCurrentOrder chan int) {
// 	// fmt.Println("ButtonPresses: ", buttonPresses)
// 	array := buttonPresses
// 	for i := 0; i < definitions.N_FLOORS; i++ {
// 		if array[i] == 1 {
// 			// fmt.Println("Going to floorfdsf: ", i)
// 			elevator.GoToFloor(i, &elevatorState, stopCurrentOrder)
// 			// fmt.Println("goToFloor Ended", i, " ended")
// 		}
// 	}
// }