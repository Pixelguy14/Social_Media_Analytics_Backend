package models

// User represents the system user.
type User struct {
	// ID is the Firestore Document ID.
	// firestore:"-" ensures it isn't stored as a field inside the document.
	ID string `json:"id" firestore:"-"`

	Name     string `json:"name" firestore:"name"`
	Email    string `json:"email" firestore:"email"`
	Username string `json:"username" firestore:"username"`
	Role     string `json:"role" firestore:"role"`
	Color    string `json:"color" firestore:"color"`

	// json:"-" prevents the password from being sent in API responses.
	// firestore:"password" stores the hashed value in the database.
	Password string `json:"-" firestore:"password"`
}
