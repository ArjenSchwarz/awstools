# Go rules

- A typed-nil pointer stored in an interface is not a nil interface. A type switch can match its concrete pointer type, then dereferencing the matched value panics. Check the matched pointer before reading its fields. This matters for AWS SDK union member interfaces.
