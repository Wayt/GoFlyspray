# GoFlyspray

GoFlyspray is a client library for Flypray bugtracker. It allow you to auth & post a new task.

## Usage

```
package main

import (
    "github.com/wayt/goflyspray"
    "log"
)

func main() {
    session, err := Endpoint("http://your.flyspray.com").Auth("username", "password")
    if err != nil {
        log.Fatal(err)
    }

    form := DefaultNewTaskForm()
    form.ItemSummary = "Title"
    form.DetailedDesc = "Body content"

    session.NewTask(form)
}
```
