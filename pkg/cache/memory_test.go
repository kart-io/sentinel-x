package cache

import (
	"sort"
	"sync"
	"testing"
)

type User struct {
	ID   int
	Name string
	Role string
	Age  int
}

func TestMemoryCache_Basic(t *testing.T) {
	c := NewMemoryCache[int, User]()

	u1 := User{ID: 1, Name: "Alice", Role: "Admin", Age: 30}
	c.Set(1, u1)

	// Get
	if got, ok := c.Get(1); !ok || got != u1 {
		t.Errorf("Get(1) = %v, %v; want %v, true", got, ok, u1)
	}

	// Contains
	if !c.Contains(1) {
		t.Error("Contains(1) = false, want true")
	}

	// Len
	if c.Len() != 1 {
		t.Errorf("Len() = %d, want 1", c.Len())
	}

	// Del
	c.Del(1)
	if _, ok := c.Get(1); ok {
		t.Error("Get(1) found item after Del")
	}
	if c.Len() != 0 {
		t.Errorf("Len() = %d, want 0", c.Len())
	}
}

func TestMemoryCache_Load(t *testing.T) {
	c := NewMemoryCache[int, User]()
	c.AddIndex("role", func(u User) any { return u.Role })

	users := []User{
		{ID: 1, Name: "User1", Role: "Entry"},
		{ID: 2, Name: "User2", Role: "Entry"},
		{ID: 3, Name: "User3", Role: "Admin"},
	}

	c.Load(users, func(u User) int { return u.ID })

	if c.Len() != 3 {
		t.Errorf("Len() = %d, want 3", c.Len())
	}

	entries, _ := c.Find("role", "Entry")
	if len(entries) != 2 {
		t.Errorf("Find(role, Entry) returned %d items, want 2", len(entries))
	}
}

func TestMemoryCache_Indexing(t *testing.T) {
	c := NewMemoryCache[int, User]()

	// Add Indexes
	c.AddIndex("role", func(u User) any { return u.Role })
	c.AddIndex("age", func(u User) any { return u.Age })

	users := []User{
		{ID: 1, Name: "Alice", Role: "Admin", Age: 30},
		{ID: 2, Name: "Bob", Role: "User", Age: 25},
		{ID: 3, Name: "Charlie", Role: "User", Age: 30},
	}

	for _, u := range users {
		c.Set(u.ID, u)
	}

	// Test Find by Role
	admins, err := c.Find("role", "Admin")
	if err != nil {
		t.Fatalf("Find(role, Admin) error: %v", err)
	}
	if len(admins) != 1 || admins[0].ID != 1 {
		t.Errorf("Find(role, Admin) = %v, want [Alice]", admins)
	}

	regularUsers, err := c.Find("role", "User")
	if err != nil {
		t.Fatalf("Find(role, User) error: %v", err)
	}
	if len(regularUsers) != 2 {
		t.Errorf("Find(role, User) returned %d items, want 2", len(regularUsers))
	}

	// Test Find by Age
	thirtyYearOlds, err := c.Find("age", 30)
	if err != nil {
		t.Fatalf("Find(age, 30) error: %v", err)
	}
	if len(thirtyYearOlds) != 2 {
		t.Errorf("Find(age, 30) returned %d items, want 2", len(thirtyYearOlds))
	}

	// Test Update affecting Index
	// Change Alice to User
	users[0].Role = "User"
	c.Set(1, users[0])

	admins, _ = c.Find("role", "Admin")
	if len(admins) != 0 {
		t.Errorf("Find(role, Admin) after update = %v, want []", admins)
	}

	regularUsers, _ = c.Find("role", "User")
	if len(regularUsers) != 3 {
		t.Errorf("Find(role, User) after update returned %d items, want 3", len(regularUsers))
	}

	// Test Delete affecting Index
	c.Del(1)
	regularUsers, _ = c.Find("role", "User")
	if len(regularUsers) != 2 {
		t.Errorf("Find(role, User) after delete returned %d items, want 2", len(regularUsers))
	}
}

func TestMemoryCache_Filter(t *testing.T) {
	c := NewMemoryCache[int, User]()
	for i := 0; i < 10; i++ {
		c.Set(i, User{ID: i, Age: i * 5})
	}

	// Filter users older than 20
	res := c.Filter(func(u User) bool {
		return u.Age > 20
	})

	// Expect 25, 30, 35, 40, 45 (5 items)
	if len(res) != 5 {
		t.Errorf("Filter(>20) returned %d items, want 5", len(res))
	}
}

func TestMemoryCache_Concurrency(_ *testing.T) {
	c := NewMemoryCache[int, int]()
	c.AddIndex("mod", func(v int) any { return v % 2 })

	var wg sync.WaitGroup
	workers := 10
	ops := 100

	// Writers
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				key := id*ops + i
				c.Set(key, key)
				if i%2 == 0 {
					c.Del(key)
				}
			}
		}(w)
	}

	// Readers
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				c.Get(1)
				_, _ = c.Find("mod", 0)
			}
		}()
	}

	wg.Wait()
}

func TestMemoryCache_KeysValues(t *testing.T) {
	c := NewMemoryCache[string, int]()
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)

	keys := c.Keys()
	values := c.Values()

	if len(keys) != 3 {
		t.Errorf("Keys() len = %d, want 3", len(keys))
	}
	if len(values) != 3 {
		t.Errorf("Values() len = %d, want 3", len(values))
	}

	sort.Strings(keys)
	if keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Errorf("Keys() returned unexpected: %v", keys)
	}
}
