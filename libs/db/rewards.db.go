// libs/db/rewards.db.go

package db

import (
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
)

// AddTaskForUser adds a task ID for a given user address if it doesn't exist already
func AddTaskForUser(userAddress string, taskID int) error {
	var task RewardTable
	result := db.Where("user_address = ?", userAddress).First(&task)
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return result.Error
	}

	if result.Error == gorm.ErrRecordNotFound {
		// User record not found, create new record with the task ID
		task = RewardTable{
			UserAddress:    userAddress,
			CompletedTasks: toJSONSlice([]int{taskID}),
			DailyPoints:    0, // Initialize with 0
		}
		err := db.Create(&task).Error
		if err != nil {
			return err
		}
	} else {
		// User record found, update existing record with new task ID if not already present
		var completedTasks []int
		if err := json.Unmarshal(task.CompletedTasks, &completedTasks); err != nil {
			return fmt.Errorf("failed to unmarshal completed tasks: %v", err)
		}

		// Check if taskID already exists in completedTasks
		found := false
		for _, id := range completedTasks {
			if id == taskID {
				found = true
				break
			}
		}

		// If taskID not found, append it to completedTasks and update database
		if !found {
			completedTasks = append(completedTasks, taskID)
			task.CompletedTasks = toJSONSlice(completedTasks)
			err := db.Save(&task).Error
			if err != nil {
				return err
			}
		} else {
			// Task ID already exists, do nothing (or handle as needed)
			fmt.Printf("Task ID %d already exists for user %s\n", taskID, userAddress)
		}
	}

	return nil
}
func UpdateDailyPointsForUser(userAddress string, points int) error {
	var task RewardTable
	result := db.Where("user_address = ?", userAddress).First(&task)
	if result.Error != nil {
		return result.Error
	}

	// Update the DailyPoints field
	task.DailyPoints = points
	err := db.Save(&task).Error
	if err != nil {
		return err
	}

	return nil
}

// toJSONSlice converts a slice of integers to a JSON-encoded byte slice
func toJSONSlice(tasks []int) []byte {
	tasksBytes, _ := json.Marshal(tasks) // error handling omitted for brevity
	return tasksBytes
}

// GetCompletedTasksForUser retrieves all completed task IDs for a user address
func GetCompletedTasksForUser(subaccountIdHex string) ([]int, int, error) {
	var task RewardTable
	result := db.Where("user_address = ?", subaccountIdHex).First(&task)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Return an empty array if no record is found
			return []int{}, 0, nil
		}
		return nil, 0, result.Error
	}

	var completedTasks []int
	if err := json.Unmarshal(task.CompletedTasks, &completedTasks); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal completed tasks: %v", err)
	}

	return completedTasks, task.DailyPoints, nil
}
