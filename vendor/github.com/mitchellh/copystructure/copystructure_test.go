package copystructure

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestCopy_complex(t *testing.T) {
	v := map[string]interface{}{
		"foo": []string{"a", "b"},
		"bar": "baz",
	}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestCopy_primitive(t *testing.T) {
	cases := []interface{}{
		42,
		"foo",
		1.2,
	}

	for _, tc := range cases {
		result, err := Copy(tc)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		if result != tc {
			t.Fatalf("bad: %#v", result)
		}
	}
}

func TestCopy_primitivePtr(t *testing.T) {
	cases := []interface{}{
		42,
		"foo",
		1.2,
	}

	for _, tc := range cases {
		result, err := Copy(&tc)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if !reflect.DeepEqual(result, &tc) {
			t.Fatalf("bad: %#v", result)
		}
	}
}

func TestCopy_map(t *testing.T) {
	v := map[string]interface{}{
		"bar": "baz",
	}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestCopy_slice(t *testing.T) {
	v := []string{"bar", "baz"}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestCopy_struct(t *testing.T) {
	type test struct {
		Value string
	}

	v := test{Value: "foo"}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestCopy_structPtr(t *testing.T) {
	type test struct {
		Value string
	}

	v := &test{Value: "foo"}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestCopy_structNil(t *testing.T) {
	type test struct {
		Value string
	}

	var v *test
	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if v, ok := result.(*test); !ok {
		t.Fatalf("bad: %#v", result)
	} else if v != nil {
		t.Fatalf("bad: %#v", v)
	}
}

func TestCopy_structNested(t *testing.T) {
	type TestInner struct{}

	type Test struct {
		Test *TestInner
	}

	v := Test{}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestCopy_structUnexported(t *testing.T) {
	type test struct {
		Value string

		private string
	}

	v := test{Value: "foo"}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestCopy_nestedStructUnexported(t *testing.T) {
	type subTest struct {
		mine string
	}

	type test struct {
		Value   string
		private subTest
	}

	v := test{Value: "foo"}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestCopy_time(t *testing.T) {
	type test struct {
		Value time.Time
	}

	v := test{Value: time.Now().UTC()}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

func TestCopy_aliased(t *testing.T) {
	type (
		Int   int
		Str   string
		Map   map[Int]interface{}
		Slice []Str
	)

	v := Map{
		1: Map{10: 20},
		2: Map(nil),
		3: Slice{"a", "b"},
	}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

type EmbeddedLocker struct {
	sync.Mutex
	Map map[int]int
}

func TestCopy_embeddedLocker(t *testing.T) {
	v := &EmbeddedLocker{
		Map: map[int]int{42: 111},
	}
	// start locked to prevent copying
	v.Lock()

	var result interface{}
	var err error

	copied := make(chan bool)

	go func() {
		result, err = Config{Lock: true}.Copy(v)
		close(copied)
	}()

	// pause slightly to make sure copying is blocked
	select {
	case <-copied:
		t.Fatal("copy completed while locked!")
	case <-time.After(100 * time.Millisecond):
		v.Unlock()
	}

	<-copied

	// test that the mutex is in the correct state
	result.(*EmbeddedLocker).Lock()
	result.(*EmbeddedLocker).Unlock()

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

// this will trigger the race detector, and usually panic if the original
// struct isn't properly locked during Copy
func TestCopy_lockRace(t *testing.T) {
	v := &EmbeddedLocker{
		Map: map[int]int{},
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				v.Lock()
				v.Map[i] = i
				v.Unlock()
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			Config{Lock: true}.Copy(v)
		}()
	}

	wg.Wait()
	result, err := Config{Lock: true}.Copy(v)

	// test that the mutex is in the correct state
	result.(*EmbeddedLocker).Lock()
	result.(*EmbeddedLocker).Unlock()

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

type LockedField struct {
	String string
	Locker *EmbeddedLocker
	// this should not get locked or have its state copied
	Mutex    sync.Mutex
	nilMutex *sync.Mutex
}

func TestCopy_lockedField(t *testing.T) {
	v := &LockedField{
		String: "orig",
		Locker: &EmbeddedLocker{
			Map: map[int]int{42: 111},
		},
	}

	// start locked to prevent copying
	v.Locker.Lock()
	v.Mutex.Lock()

	var result interface{}
	var err error

	copied := make(chan bool)

	go func() {
		result, err = Config{Lock: true}.Copy(v)
		close(copied)
	}()

	// pause slightly to make sure copying is blocked
	select {
	case <-copied:
		t.Fatal("copy completed while locked!")
	case <-time.After(100 * time.Millisecond):
		v.Locker.Unlock()
	}

	<-copied

	// test that the mutexes are in the correct state
	result.(*LockedField).Locker.Lock()
	result.(*LockedField).Locker.Unlock()
	result.(*LockedField).Mutex.Lock()
	result.(*LockedField).Mutex.Unlock()

	// this wasn't  blocking, but should be unlocked for DeepEqual
	v.Mutex.Unlock()

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("expected:\n%#v\nbad:\n%#v\n", v, result)
	}
}

// test something that doesn't contain a lock internally
type lockedMap map[int]int

var mapLock sync.Mutex

func (m lockedMap) Lock()   { mapLock.Lock() }
func (m lockedMap) Unlock() { mapLock.Unlock() }

func TestCopy_lockedMap(t *testing.T) {
	v := lockedMap{1: 2}
	v.Lock()

	var result interface{}
	var err error

	copied := make(chan bool)

	go func() {
		result, err = Config{Lock: true}.Copy(&v)
		close(copied)
	}()

	// pause slightly to make sure copying is blocked
	select {
	case <-copied:
		t.Fatal("copy completed while locked!")
	case <-time.After(100 * time.Millisecond):
		v.Unlock()
	}

	<-copied

	// test that the mutex is in the correct state
	result.(lockedMap).Lock()
	result.(lockedMap).Unlock()

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

// Use an RLock if available
type RLocker struct {
	sync.RWMutex
	Map map[int]int
}

func TestCopy_rLocker(t *testing.T) {
	v := &RLocker{
		Map: map[int]int{1: 2},
	}
	v.Lock()

	var result interface{}
	var err error

	copied := make(chan bool)

	go func() {
		result, err = Config{Lock: true}.Copy(v)
		close(copied)
	}()

	// pause slightly to make sure copying is blocked
	select {
	case <-copied:
		t.Fatal("copy completed while locked!")
	case <-time.After(100 * time.Millisecond):
		v.Unlock()
	}

	<-copied

	// test that the mutex is in the correct state
	vCopy := result.(*RLocker)
	vCopy.Lock()
	vCopy.Unlock()
	vCopy.RLock()
	vCopy.RUnlock()

	// now make sure we can copy during an RLock
	v.RLock()
	result, err = Config{Lock: true}.Copy(v)
	if err != nil {
		t.Fatal(err)
	}
	v.RUnlock()

	vCopy = result.(*RLocker)
	vCopy.Lock()
	vCopy.Unlock()
	vCopy.RLock()
	vCopy.RUnlock()

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("bad: %#v", result)
	}
}

// Test that we don't panic when encountering nil Lockers
func TestCopy_missingLockedField(t *testing.T) {
	v := &LockedField{
		String: "orig",
	}

	result, err := Config{Lock: true}.Copy(v)

	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("expected:\n%#v\nbad:\n%#v\n", v, result)
	}
}

func TestCopy_sliceWithNil(t *testing.T) {
	v := [](*int){nil}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("expected:\n%#v\ngot:\n%#v", v, result)
	}
}

func TestCopy_mapWithNil(t *testing.T) {
	v := map[int](*int){0: nil}

	result, err := Copy(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(result, v) {
		t.Fatalf("expected:\n%#v\ngot:\n%#v", v, result)
	}
}

// While this is safe to lock and copy directly, copystructure requires a
// pointer to reflect the value safely.
func TestCopy_valueWithLockPointer(t *testing.T) {
	v := struct {
		*sync.Mutex
		X int
	}{
		Mutex: &sync.Mutex{},
		X:     3,
	}

	_, err := Config{Lock: true}.Copy(v)

	if err != errPointerRequired {
		t.Fatalf("expected errPointerRequired, got: %v", err)
	}
}

func TestCopy_mapWithPointers(t *testing.T) {
	type T struct {
		S string
	}
	v := map[string]interface{}{
		"a": &T{S: "hello"},
	}

	result, err := Copy(v)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(v, result) {
		t.Fatal(result)
	}
}

func TestCopy_structWithMapWithPointers(t *testing.T) {
	type T struct {
		S string
		M map[string]interface{}
	}
	v := &T{
		S: "a",
		M: map[string]interface{}{
			"b": &T{
				S: "b",
			},
		},
	}

	result, err := Copy(v)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(v, result) {
		t.Fatal(result)
	}
}
