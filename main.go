package main

import (
	"fmt"
	"myapp/data"
	"myapp/handlers"
	"myapp/middleware"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/techarm/celeritas"
)

type application struct {
	App        *celeritas.Celeritas
	Middleware *middleware.Middleware
	Handlers   *handlers.Handlers
	Models     data.Models
	wg         sync.WaitGroup
}

func main() {
	c := initApplication()
	go c.listenFroShutdown()

	err := c.App.ListenAndServe()
	if err != nil {
		c.App.ErrorLog.Println(err)
	}
}

func (a *application) shutdown() {
	// put any clean up task
	fmt.Println("do clean up task")

	// block until the WaitGroup is empty
	a.wg.Wait()
}

func (a *application) listenFroShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	s := <-quit

	a.App.InfoLog.Println("Received signal", s.String())
	a.shutdown()

	os.Exit(0)
}
