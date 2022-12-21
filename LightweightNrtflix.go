package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

// User represents a user in the database
type User struct {
	ID       int
	FullName string
	Age      int
	Email    string
	Password string
}

// Movie represents a movie in the database
type Movie struct {
	ID          int 
	Name        string
	Description string
	Date        string
	Cover       string
	UserID      int
}

// Rating represents a rating for a movie in the database
type Rating struct {
	ID     int
	Movie  int
	User   int
	Rating int
	Review string
}

func main() {
	// Connect to the database
	db, err := sql.Open("mysql", "mht67983:2txLIauE19eN2Ra5@/bludb")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize the Gin web framework
	r := gin.Default()

	// Define the API routes
	r.POST("/register", func(c *gin.Context) {
		// Bind the form data to a User struct
		var user User
		if err := c.Bind(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Insert the new user into the database
		result, err := db.Exec("INSERT INTO users (full_name, age, email, password) VALUES (?, ?, ?, ?)", user.FullName, user.Age, user.Email, user.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Get the ID of the newly inserted user
		user.ID, err = result.LastInsertId()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, user)
	})

	r.POST("/login", func(c *gin.Context) {
		// Bind the form data to a User struct
		var user User
		if err := c.Bind(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if the email and password match a user in the database
		var dbUser User
		err := db.QueryRow("SELECT id, full_name, age, email, password FROM users WHERE email = ? AND password = ?", user.Email, user.Password).Scan(&dbUser.ID, &dbUser.FullName, &dbUser.Age, &dbUser.Email,&dbUser.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Email or password is incorrect"})
			return
		}

		// Generate and return an authentication token
		token, err := generateToken(dbUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	r.POST("/add-movie", func(c *gin.Context) {
		// Bind the form data to a Movie struct
		var movie Movie
		if err := c.Bind(&movie); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get the user ID from the authentication token
		userID, err := getUserIDFromToken(c.GetHeader("Authorization"))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Set the user ID on the movie object
		movie.UserID = userID

		// Insert the new movie into the database
		result, err := db.Exec("INSERT INTO movies (name, description, date, cover, user_id) VALUES (?, ?, ?, ?, ?)", movie.Name, movie.Description, movie.Date, movie.Cover, movie.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Get the ID of the newly inserted movie
		movie.ID, err = result.LastInsertId()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, movie)
	})

	r.PUT("/edit-movie/:id", func(c *gin.Context) {
		// Get the movie ID from the URL params
		movieID := c.Param("id")

		// Bind the form data to a Movie struct
		var movie Movie
		if err := c.Bind(&movie); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get the user ID from the authentication token
		userID, err := getUserIDFromToken(c.GetHeader("Authorization"))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Check if the user is the owner of the movie
		var dbMovie Movie
		err = db.QueryRow("SELECT id, user_id FROM movies WHERE id = ?", movieID).Scan(&dbMovie.ID, &dbMovie.UserID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Movie not found"})
			return
		}
		if dbMovie.UserID != userID {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not the owner of this movie"})
			return
		}

		// Update the movie in the database
		_, err = db.Exec("UPDATE movies SET name = ?, description = ?, date = ?, cover = ? WHERE id = ?", movie.Name, movie.Description, movie.Date, movie.Cover, movieID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusOK)
	})

	r.DELETE("/delete-movie/:id", func(c *gin.Context) {
		// Get the movie ID from the URL params
		movieID := c.Param("id")

		// Get the user ID from the authentication token
		userID, err := getUserIDFromToken(c.GetHeader("Authorization"))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Check if the user is the owner of the movie
		var dbMovie Movie
		err = db.QueryRow("SELECT id, user_id FROM movies WHERE id = ?", movieID).Scan(&dbMovie.ID, &dbMovie.UserID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Movie not found"})
			return
		}
		if dbMovie.UserID != userID {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "You are not the owner of this movie"})
			return
		}

		// Delete the movie from the database
		_, err = db.Exec("DELETE FROM movies WHERE id = ?", movieID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}		c.Status(http.StatusOK)
	})

	r.GET("/get-movie-info/:id", func(c *gin.Context) {
		// Get the movie ID from the URL params
		movieID := c.Param("id")

		// Retrieve the movie from the database
		var movie Movie
		err := db.QueryRow("SELECT id, name, description, date, cover FROM movies WHERE id = ?", movieID).Scan(&movie.ID, &movie.Name, &movie.Description, &movie.Date, &movie.Cover)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Movie not found"})
			return
		}

		// Retrieve the ratings for the movie
		rows, err := db.Query("SELECT rating FROM ratings WHERE movie = ?", movieID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		// Calculate the average rating
		var ratings []int
		for rows.Next() {
			var rating int
			if err := rows.Scan(&rating); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			ratings = append(ratings, rating)
		}

		var sum int
		for _, rating := range ratings {
			sum += rating
		}
		averageRating := sum / len(ratings)

		c.JSON(http.StatusOK, gin.H{
			"id":           movie.ID,
			"name":         movie.Name,
			"description":  movie.Description,
			"date":         movie.Date,
			"cover":        movie.Cover,
			"averageRating": averageRating,
		})
	})

	r.GET("/list-movies", func(c *gin.Context) {
		// Retrieve the movies from the database
		rows, err := db.Query("SELECT id, name, description, date, cover FROM movies")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var movies []Movie
		for rows.Next() {
			var movie Movie
			if err := rows.Scan(&movie.ID, &movie.Name, &movie.Description, &movie.Date, &movie.Cover); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			movies = append(movies, movie)
		}

		// Check if the user wants to sort the movies by date or rating
		sortBy := c.Query("sortBy")
		if sortBy == "date" {
			// Sort the movies by date
			sort.Slice(movies, func(i, j int) bool {
				return movies[i].Date < movies[j].Date
			})
		} else if sortBy == "rating" {
			// Sort the movies by rating
			sort.Slice(movies, func(i, j int) bool {
				return movies[i].AverageRating > movies[j].AverageRating
			})
		}

		c.JSON(http.StatusOK, movies)
	})

	r.POST("/add-movie-to-watched-list", func(c *gin.Context) {
		// Bind the form data to a Movie struct
		var movie Movie
		if err := c.Bind(&movie); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get the user ID from the authentication token
		userID, err := getUserIDFromToken(c.GetHeader("Authorization"))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Add the movie to the user's watched list in the database
		_, err = db.Exec("INSERT INTO watched (user, movie) VALUES (?, ?)", userID, movie.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusOK)
	})

	r.POST("/rate-and-review-movie", func(c *gin.Context) {
		// Bind the form data to a Rating struct
		var rating Rating
		if err := c.Bind(&rating); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get the user ID from the authentication token
		userID, err := getUserIDFromToken(c.GetHeader("Authorization"))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Check if the movie is in the user's watched list
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM watched WHERE user = ? AND movie = ?", userID, rating.MovieID).Scan(&count)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if count == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "You must have watched the movie before rating it"})
			return
		}

		// Update the rating and review for the movie in the database
		_, err = db.Exec("UPDATE ratings SET rating = ?, review = ? WHERE user = ? AND movie = ?", rating.Rating, rating.Review, userID, rating.MovieID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusOK)
	})

	// Start the server
	r.Run()
}



