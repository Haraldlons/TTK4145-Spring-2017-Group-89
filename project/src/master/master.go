package master

import (
	"../definitions"
	"../watchdog"
	// "../driver"
	"../network"
	// "../storage"
	"fmt"
	"math"
	"os/exec"
	// "string"
	// "math/rand"
	// "net"
	"sync"
	"time"
)

func Run() {
	fmt.Println("I'm a MASTER!")


	// Spawn new "personal" slaves
	newSlave := exec.Command("gnome-terminal", "-x", "sh", "-c", "go run main.go")
	newSlave.Run()

	// Might change when network is unplugged. THIS MUST BE HANDLED!!!
	// TODO
	master_id, _ := network.GetLocalIP()
	mutex := &sync.Mutex{}

	// Channel for passing totalOrderList from listener->handler->sender
	totalOrderListChan := make(chan definitions.Elevators) // Channel for passing totalOrderList

	// Channel for sending kill-signal to all network-related goroutines
	stopSendingChan := make(chan bool)

	// Channels for sending a map with alive-status of all slaves connected to the network
	allSlavesAliveMapChanMap := map[string]chan map[string]bool{
		// "toKeepTrackOfAllAliveSlaves": make(chan map[string]bool),
		"toRun":                     make(chan map[string]bool),
		"toHandleUpdatesFromSlaves": make(chan map[string]bool),
	}


	// Channels for sending the id of a slave from the listener to others
	updatedSlaveIdChanMap := map[string]chan string{
		"toWatchdog": make(chan string),
		// "toKeepTrackOfAllAliveSlaves": make(chan string),
	}


	// Send alive messages from master regularly
	go network.SendMasterIsAliveRegularly(master_id, stopSendingChan)

	// Listen after alive slaves and keep track of alive ones
	go network.ListenAfterAliveSlavesRegularly(updatedSlaveIdChanMap, stopSendingChan)
	go watchdog.KeepTrackOfAllAliveSlaves(updatedSlaveIdChanMap["toWatchdog"], allSlavesAliveMapChanMap)

	// Receive messages from slaves, handle, then send to all slaves
	go handleUpdatesFromSlaves(totalOrderListChan, allSlavesAliveMapChanMap["toHandleUpdatesFromSlaves"], master_id, mutex)
	go sendMessageToSlavesOnUpdate(totalOrderListChan, mutex)

	for {
		select {
		// Blocking statement to listen for changes in slaves' status
		case allSlavesAliveMap := <-allSlavesAliveMapChanMap["toRun"]:
			// If any slaves died, their last known orders will be redistributed to alive slaves
			go redistributeOrders(allSlavesAliveMap, totalOrderListChan, master_id)
		}
	}
}

// Update order list in "orders" object with the command defined by externalButtonPress
func updateOrders(orders *definitions.Orders, externalButtonPress definitions.Order, elevatorState definitions.ElevatorState) {
	if externalButtonPress.Direction == definitions.DIR_STOP {
		/*Detected internal button press*/
		distributeInternalOrderToOrderList(externalButtonPress, orders, elevatorState)
	}
	if checkForDuplicateOrder(orders, externalButtonPress.Floor) { // TODO: DO NOT REMOVE ORDERS ALONG THE SAME DIRECTION
		// fmt.Println("This order is already in the queue!")
		// fmt.Println("\nORDERS BEFORE findAndReplaceOrderIfSameDirection():", orders)
		findAndReplaceOrderIfSameDirection(orders, externalButtonPress, elevatorState.Direction) //TODO
		// fmt.Println("ORDERS AFTER findAndReplaceOrderIfSameDirection():", orders, "\n")
		return
	}

	// fmt.Println("Orders received by updateOrders():", orders)
	// fmt.Println("Elevatorstate received by updateOrders():", elevatorState)
	// fmt.Println("ExternalButtonPress received by updateOrders():", externalButtonPress)

	if len(orders.Orders) > 0 { // For safety
		// Check to see if order should be placed first based on current elevator state
		if elevatorState.Direction == externalButtonPress.Direction && FloorIsInbetween(orders.Orders[0].Floor, externalButtonPress.Floor, elevatorState.LastFloor, elevatorState.Direction) {
			// Insert Order in first position
			// fmt.Println("Inserting order in first postion")

			orders.Orders = append(orders.Orders, definitions.Order{})
			copy(orders.Orders[1:], orders.Orders[:])
			orders.Orders[0] = externalButtonPress
			// fmt.Println("Orders returned by updateOrders():", orders)
			return
		}

	}

	for i := 1; i < len(orders.Orders); i++ {
		direction := orders.Orders[i].Direction
		if externalButtonPress.Direction == direction { // Elevator is moving in the right direction
			switch direction {
			case definitions.DIR_UP:
				if externalButtonPress.Floor < orders.Orders[i].Floor {
					// Insert Order in position (i)
					// fmt.Println("Inserting order in postion", i)
					orders.Orders = append(orders.Orders, definitions.Order{})
					copy(orders.Orders[i+1:], orders.Orders[i:])
					orders.Orders[i] = externalButtonPress
					// fmt.Println("Orders returned by updateOrders():", orders)
					return
				}
			case definitions.DIR_DOWN:
				if externalButtonPress.Floor > orders.Orders[i].Floor {
					// Insert Order in position (i+1)
					// fmt.Println("Inserting order in postion", i)

					orders.Orders = append(orders.Orders, definitions.Order{})
					copy(orders.Orders[i+1:], orders.Orders[i:])
					orders.Orders[i] = externalButtonPress
					// fmt.Println("Orders returned by updateOrders():", orders)
					return

				}
			default:
				fmt.Println("Something weird is up, buddy")
			}
		}
	}
	// Place order at back of orderList
	// fmt.Println("Placing order at back of order list")
	orders.Orders = append(orders.Orders, externalButtonPress)
	// fmt.Println("Orders returned by updateOrders():", orders)
}

// Don't accept more orders to same floor. Assume every person gets on elevator.
func checkForDuplicateOrder(orders *definitions.Orders, buttonPressedFloor int) bool {
	for _, order := range orders.Orders {
		if order.Floor == buttonPressedFloor {
			return true
		}
	}
	return false
}

func findAndReplaceOrderIfSameDirection(orders *definitions.Orders, externalButtonPress definitions.Order, elevatorDirection int) {
	// No point if orderList only has one order
	if len(orders.Orders) > 1 { 
		return
	}
	for i:= range orders.Orders {
		// Elevator is moving in the same direction as the buttonPress
		// TODO: THE ABOVE STATEMENT IS PRETTY MUCH NEVER CORRECT
		if orders.Orders[i].Floor == externalButtonPress.Floor && externalButtonPress.Direction == elevatorDirection {
			orders.Orders[i].Direction = externalButtonPress.Direction // Change direction of order
			return
		}
	}
}

func FloorIsInbetween(orderFloor int, buttonFloor int, elevatorLastFloor int, elevatorDirection int) bool {
	switch elevatorDirection {
	case definitions.DIR_UP:
		return buttonFloor > elevatorLastFloor &&
			buttonFloor < orderFloor
	case definitions.DIR_DOWN:
		return buttonFloor < elevatorLastFloor &&
			buttonFloor > orderFloor
	default:
		fmt.Println("Something is wrong in floorIsBetween()")
		return false
	}
}

// Returns id of best suited elevator, assuming elevatorStates is a map with ids
func findLowestCostElevator(elevatorStates map[string]definitions.ElevatorState, externalButtonPress definitions.Order, master_id string) string {
	minCost := 2 * definitions.N_FLOORS
	bestElevator_id := master_id
	destinationFloor := externalButtonPress.Floor
	destinationDirection := externalButtonPress.Direction

	for id, elevatorState := range elevatorStates { // Loop through map
		travelDirection := findTravelDirection(elevatorState.LastFloor, destinationFloor)
		tempCost := int(math.Abs(float64(destinationFloor - elevatorState.LastFloor)))

		if elevatorState.Destination == definitions.IDLE {
			// Elevator is idle
			tempCost = tempCost - 1 // Prioritize idle elevators
		} else if elevatorState.Destination != destinationFloor {
			// No additional cost if elevator destination is the same as order destination
			if elevatorHasAdditionalCost(travelDirection, destinationFloor, destinationDirection, elevatorState) {
				costToDest := int(math.Abs(float64(elevatorState.Destination - elevatorState.LastFloor)))
				tempCost = costToDest + int(math.Abs(float64(destinationFloor-elevatorState.Destination)))
				fmt.Println("Elevator with identifier", id, " has extra cost")
			}
		}

		if tempCost < minCost {
			minCost = tempCost
			bestElevator_id = id
		}
		// fmt.Println("Cost of elevator", id, ":", tempCost)
	}
	// fmt.Println("Minimum cost:", minCost)
	return bestElevator_id
}

func findTravelDirection(startFloor int, destinationFloor int) int {
	if destinationFloor > startFloor {
		return definitions.DIR_UP
	} else if destinationFloor == startFloor {
		return definitions.DIR_UP
	} else {
		return definitions.DIR_DOWN
	}
}

// Returns true if elevator passes destinationFloor on it's way to elevatorDestination
func elevatorPassesDestinationFloor(travelDirection int, destinationFloor int, elevatorDestination int) bool {
	return (travelDirection == definitions.DIR_UP && destinationFloor-elevatorDestination < 0) ||
		(travelDirection == definitions.DIR_DOWN && destinationFloor-elevatorDestination > 0)
}

// Returns true if elevator can not go straight to destinationFloor
func elevatorHasAdditionalCost(travelDirection int, destinationFloor int, destinationDirection int, elevState definitions.ElevatorState) bool {
	return (elevatorPassesDestinationFloor(travelDirection, destinationFloor, elevState.Destination) &&
		travelDirection != destinationDirection) || // Elevator is traveling in the opposite direction of Order
		travelDirection != elevState.Direction || // Elevator is moving in the opposite direction relative to destination
		destinationFloor == elevState.LastFloor // Elevator has probably passed destination
}

func handleUpdatesFromSlaves(totalOrderListChan chan definitions.Elevators, allSlavesAliveMapChanMap chan map[string]bool, elevator_id string, mutex *sync.Mutex) {
	// Initialize local channel
	msgChan := make(chan definitions.MSG_to_master)

	// Initialize totalOrderList
	totalOrderList := definitions.Elevators{}

	// Initialize maps in totalOrderList
	totalOrderList.OrderMap = make(map[string]definitions.Orders)
	totalOrderList.ElevatorStateMap = make(map[string]definitions.ElevatorState)

	// Initialize map of aliveSlaves
	allSlavesAliveMap := make(map[string]bool)

	

	// Start goroutine to listen for updates from slaves
	go network.ListenToSlave(msgChan)

	for {
		select {
		case msg := <-msgChan: // New message received
			// fmt.Println("List of orders from msgChan:", msg.Orders)
			// fmt.Println("List of orders from totalOrderList:", totalOrderList.OrderMap[msg.Id])
			// Update totalOrderList with information from message
			fmt.Println("---------------------------------")
			mutex.Lock()
			fmt.Println("Message received from slave:", msg)
			mutex.Unlock()

			mutex.Lock()
			totalOrderList.OrderMap[msg.Id] = msg.Orders
			totalOrderList.ElevatorStateMap[msg.Id] = msg.ElevatorState
			mutex.Unlock()

			// Get map of states
			elevatorStateMap := totalOrderList.ElevatorStateMap
			bestElevator_id := ""

			// Find elevator best suited for taking the received orders, and add orders to corresponding order lists
			for _, externalButtonPress := range msg.ExternalButtonPresses {

				mutex.Lock() // For accesing maps
				if externalButtonPress.Direction == definitions.DIR_STOP {
					// Actually an internal button press. Has to be executed by sender
					bestElevator_id = msg.Id
					fmt.Println("Internal button press. Best elevator is sender:", bestElevator_id)
				} else {
					bestElevator_id = findLowestCostElevator(elevatorStateMap, externalButtonPress, elevator_id)
				}
				mutex.Unlock()

				fmt.Println("Best elevator:", bestElevator_id, ", for order", externalButtonPress)

				mutex.Lock()
				orders := totalOrderList.OrderMap[bestElevator_id]
				updateOrders(&orders, externalButtonPress, elevatorStateMap[bestElevator_id])
				totalOrderList.OrderMap[bestElevator_id] = orders
				mutex.Unlock()
			}

			// fmt.Println("Total order list: ", totalOrderList)

			// Send updates to channel
			totalOrderListChan <- totalOrderList
			// go func() {
			// 	completedUpdateOfOrderList <- true
			// }()
			time.Sleep(time.Millisecond * 100)
		// case slavesAliveMap = <- slavesAliveMapToHandleUpdatesFromSlavesChan /*To be implemented*/
		case allSlavesAliveMap = <- allSlavesAliveMapChanMap: // Update on wether slaves are alive or not
			for slave_id, isAlive := range allSlavesAliveMap {
				// If a slave has died
				if !isAlive {
					// Delete slave from totalOrderList
					mutex.Lock()
					delete(totalOrderList.OrderMap, slave_id)
					delete(totalOrderList.ElevatorStateMap, slave_id)
					fmt.Println("TOTALORDERLIST AFTER DELETION OF DEAD SLAVE:", totalOrderList)
					mutex.Unlock()
				}
			}
		}
	}
}

// When totalorderlist is updated, send to all slaves
func sendMessageToSlavesOnUpdate(totalOrderListChan <-chan definitions.Elevators, mutex *sync.Mutex) {

	for {
		select {
		case totalOrderList := <-totalOrderListChan:
			// fmt.Println("Length of totalOrderlist: ", len(totalOrderList.OrderMap))
			if len(totalOrderList.OrderMap) != 0 {
				msg := definitions.MSG_to_slave{Elevators: totalOrderList}
				fmt.Println("Message sent to slave:", msg)
				network.SendToSlave(msg, mutex)
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// Function to be ran when program is booting.
// Used to redistribute active orders of elevators that have died
func redistributeOrders(allSlavesAliveMap map[string]bool, totalOrderListChan chan<- definitions.Elevators, master_id string) {
	totalOrderList := definitions.Elevators{}
	// storage.LoadElevatorsFromFile(&totalOrderList) //TODO

	// Loop through the id of every currently alive slave
	for id_slaves, isAlive := range allSlavesAliveMap {
		// Loop through maps of every elevator loaded from storage
		for id := range totalOrderList.OrderMap {
			if id_slaves == id && !isAlive { // Dead elevator
				orders := totalOrderList.OrderMap[id].Orders
				// Loop through every order
				for i := range orders {
					if orders[i].Direction != 0 { // Not an internal order
						// Find elevator with lowest cost, and add order to corresponding orderList
						elevator_id := findLowestCostElevator(totalOrderList.ElevatorStateMap, orders[i], master_id)
						updatedOrders := totalOrderList.OrderMap[id]
						updateOrders(&updatedOrders, orders[i], totalOrderList.ElevatorStateMap[elevator_id])
						totalOrderList.OrderMap[id] = updatedOrders
						// Send updates to channel, which in turn is sent over network
						fmt.Println("Orders have been redistributed and sent to network")
						totalOrderListChan <- totalOrderList
					}
				}
			}
		}
	}
}

// func keepTrackOfAllAliveSlaves(updatedSlaveIdChanMap map[string]chan string, allSlavesAliveMapChanMap map[string]chan map[string]bool, master_id string) {
// 	allSlavesAliveMap := make(map[string]bool)
// 	go network.ListenAfterAliveSlavesRegularly(updatedSlaveIdChanMap)
// 	for {
// 		select {
// 		// Receive status of all slaves from watchdog
// 		case allSlavesAliveMap = <-allSlavesAliveMapChanMap["toKeepTrackOfAllAliveSlaves"]:
// 			// Send status to Run()
// 			allSlavesAliveMapChanMap["toRun"] <- allSlavesAliveMap
// 		}
// 	}
// }

// Compare two IP-addresses, and return the one with the largest last three digits
func compareIdsAndReturnLargest(id_1 string, id_2 string) string {
	largest := "localhost"
	// if len(id_1) ==
	return largest
}

func distributeInternalOrderToOrderList(internalPressOrder definitions.Order, currentOrderList *definitions.Orders, elevatorState definitions.ElevatorState){

	if checkForDuplicateOrder(currentOrderList, internalPressOrder.Floor) {
		return
	}

	tempNum := 0
	if len(currentOrderList.Orders) > 0 {

		if elevatorState.Direction == 1 {
			// You are going up
			fmt.Println("You are going up")
			if currentOrderList.Orders[0].Floor == elevatorState.Destination { /* You can add in front of currentOrderList */
				fmt.Println("First order is the destination floor")
				currentOrderList.Orders = append(currentOrderList.Orders, definitions.Order{})
				copy(currentOrderList.Orders[1:], currentOrderList.Orders[:])
				currentOrderList.Orders[0] = internalPressOrder
				return
			} else { /* There are orders before destinationOrder */
				for i, order := range currentOrderList.Orders {
					fmt.Println("Order[", i, "]: ", order)
					if order.Floor > tempNum { // To check where you turn
						fmt.Println(" order.Floor > tempNum ")
						if order.Floor > internalPressOrder.Floor && elevatorState.LastFloor < internalPressOrder.Floor {
							fmt.Println("This IF STATEMENT")
							currentOrderList.Orders = append(currentOrderList.Orders, definitions.Order{})
							copy(currentOrderList.Orders[i+1:], currentOrderList.Orders[i:])
							currentOrderList.Orders[i] = internalPressOrder
							return 
						}
						tempNum = order.Floor
					}
					if tempNum == elevatorState.Destination {
						fmt.Println("tempNum == elevatorState.Destination")

						for j, order2 := range currentOrderList.Orders {
							fmt.Println("Length, ", len(currentOrderList.Orders), ", j, ", j)
							if j > i {
								if order2.Floor < internalPressOrder.Floor {
									fmt.Println("The other IF STATEMENT")

									currentOrderList.Orders = append(currentOrderList.Orders, definitions.Order{})
									copy(currentOrderList.Orders[j+1:], currentOrderList.Orders[j:])
									currentOrderList.Orders[j] = internalPressOrder
									return 
								} else if j == len(currentOrderList.Orders)-1 {
									fmt.Println("This third STATEMENT")

									currentOrderList.Orders = append(currentOrderList.Orders, definitions.Order{})
									copy(currentOrderList.Orders[j+2:], currentOrderList.Orders[j+1:])
									currentOrderList.Orders[j+1] = internalPressOrder
									return
								}
							}
						}
					}
				}
			}
		} else {
			tempNum = definitions.N_FLOORS -1
			fmt.Println("You are going down")
			if currentOrderList.Orders[0].Floor == elevatorState.Destination { /* You can add in front of currentOrderList */
				fmt.Println("First order is the destination floor")
				currentOrderList.Orders = append(currentOrderList.Orders, definitions.Order{})
				copy(currentOrderList.Orders[1:], currentOrderList.Orders[:])
				currentOrderList.Orders[0] = internalPressOrder
				return 
			} else { /* There are orders before destinationOrder */
				for i, order := range currentOrderList.Orders {
					fmt.Println("Order[", i, "]: ", order)
					if order.Floor < tempNum { // To check where you turn
						fmt.Println(" order.Floor > tempNum ")
						if order.Floor < internalPressOrder.Floor && elevatorState.LastFloor < internalPressOrder.Floor {
							fmt.Println("This IF STATEMENT")
							currentOrderList.Orders = append(currentOrderList.Orders, definitions.Order{})
							copy(currentOrderList.Orders[i+1:], currentOrderList.Orders[i:])
							currentOrderList.Orders[i] = internalPressOrder
							return 
						}
						tempNum = order.Floor
					}
					if tempNum == elevatorState.Destination {
						fmt.Println("tempNum == elevatorState.Destination")

						for j, order2 := range currentOrderList.Orders {
							fmt.Println("Length, ", len(currentOrderList.Orders), ", j, ", j)
							if j > i {
								if order2.Floor > internalPressOrder.Floor {
									fmt.Println("The other IF STATEMENT")

									currentOrderList.Orders = append(currentOrderList.Orders, definitions.Order{})
									copy(currentOrderList.Orders[j+1:], currentOrderList.Orders[j:])
									currentOrderList.Orders[j] = internalPressOrder
									return 
								} else if j == len(currentOrderList.Orders)-1 {
									fmt.Println("This third STATEMENT")

									currentOrderList.Orders = append(currentOrderList.Orders, definitions.Order{})
									copy(currentOrderList.Orders[j+2:], currentOrderList.Orders[j+1:])
									currentOrderList.Orders[j+1] = internalPressOrder
									return 
								}
							}
						}
					}
				}
			}
		}
	}else {
		currentOrderList.Orders = append(currentOrderList.Orders, internalPressOrder)
	}
}
