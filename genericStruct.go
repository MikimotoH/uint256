package uint256

import "fmt"

// Define a generic struct with type parameter T
type Container[N int] struct {
	Values [N]int64
}

// Method to get the value from the container
func (c Container[N]) GetValue(i int) int64 {
	return c.Values[i]
}

// Method to set the value in the container
func (c *Container[N]) SetValue(i int, val int64) {
	c.Values[i] = val
}

func main() {
	// Create an instance of Container with int type
	intContainer := Container[4]{}
	copy(intContainer.Values, [4]int64{42, 0, 0, 0})
	fmt.Println("Integer Value:", intContainer.GetValue(0))

	// Create an instance of Container with string type
	stringContainer := Container[4]{Values: [4]int64{0, 0, 0, 0}}
	fmt.Println("String Value:", stringContainer.GetValue(0))
}
