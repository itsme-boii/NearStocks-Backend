package models

type ExampleModel struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}
// currently unused but keeping this for a better project structure in case needed in future
func FetchAllExamples() ([]ExampleModel, error) {
    examples := []ExampleModel{
        {ID: 1, Name: "Example 1"},
        {ID: 2, Name: "Example 2"},
    }
    return examples, nil
}