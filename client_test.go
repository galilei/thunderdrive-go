package thunderdrive_test

import "github.com/galilei/thunderdrive-go"

func main() {
	c := thunderdrive.New()
	c.Login("", "")
	c.GetUsage()
	c.GetEntries("/")
	// c.Mkdir(nil, "test")
	// c.Remove([]string{""})
	// c.Upload("", "/tmp/test.txt")
}
