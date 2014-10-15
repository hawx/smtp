package smtp

import (
	"log"
)

var s *Server

func ExampleHandler() {
	var logger = func(message Message) {
		log.Printf("Recieved message from %s\n%s\n", message.Sender, message.Data)
	}

	s.Handle(logger)
}

func ExampleVerifier() {
	var users = []User{
		{"John Doe", "john.doe@example.com"},
		{"Jane Doe", "jane.doe@example.com"},
	}

	s.Verify(func(arg string) User {
		for _, user := range users {
			if user.Name == arg || user.Addr == arg {
				return user
			}
		}

		return User{}
	})
}

func ExampleServer() {
	s, err := Listen(":25", "mx.test.local")
	if err != nil {
		panic(err)
	}

	defer s.Close()

	s.Handle(func(message Message) {
		// ...
	})
}
