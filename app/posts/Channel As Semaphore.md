Channel As Semaphore | 2013-03-07
# Channel As Semaphore

---

今天看到golang-dev一个关于将channel用于semaphore的讨论(附[链接](https://groups.google.com/d/topic/golang-dev/ShqsqvCzkWg/discussion)).
通常的方法如下：

~~~ {prettyprint}

package main

import "fmt"

type Mutex chan bool
func (m Mutex) Lock() { m <- true }
func (m Mutex) Unlock() { <-m }

func Exclusive(m Mutex, i *int) {
	m.Lock()
	defer m.Unlock()
	*i++
}

func main() {
	lock := make(Mutex, 1)
	val := 0

	N := 10
	done := make(chan bool, N)
	for i := 0; i < N; i++ {
		go func() {
			defer func(){ done<- true }()
			Exclusive(lock, &val)
		}()
	}
	for i := 0; i < N; i++ { 
		<-done
	}

	fmt.Println(val)
}
~~~

但是这不满足mutex的要求：unlock必须要和后续的lock保持前后的顺序，即必须先unlock
mutex，然后才能重新去lock mutex。

为什么这个例子不能满足上述的要求呢，因为当用于mutex的chan
被填满即将unlock时，恰巧有另外一个goroutine等待lock，
这时，unlock和lock的顺序性无法被保证。
原因在于go memory model没有对此进行保证，让我们看下MM（memory
model）中对channel操作的相关规则：

> - A send on a channel happens before the corresponding receive from that
> channel completes.
> - The closing of a channel happens before a receive that returns a zero value
> because the channel is closed.
> - A receive from an unbuffered channel happens before the send on that channel
> completes.

可见，当receive 一个buffer channel时，顺序性没有进行保证。

但是，Russ Cox大牛还是有办法的，来看一下他的方法，还是使用上面的代码：

~~~ {prettyprint}

package main

import "fmt"

type Mutex chan bool
-func (m Mutex) Lock() { m <- true }
+func (m Mutex) Lock() { <-m }
-func (m Mutex) Unlock() { <-m }
+func (m Mutex) Unlock() { m <- true }

func Exclusive(m Mutex, i *int) {
	m.Lock()
	defer m.Unlock()
	*i++
}

func main() {
	lock := make(Mutex, 1)
+	lock <- true
	val := 0

	N := 10
	done := make(chan bool, N)
	for i := 0; i < N; i++ {
		go func() {
			defer func(){ done<- true }()
			Exclusive(lock, &val)
		}()
	}
	for i := 0; i < N; i++ { 
		<-done
	}

	fmt.Println(val)
}
~~~

看见不同了吧，现在lock对应的是receive from buffer channel，
而unlock对应的是 send on buffer channel,
这样unlock就可以保证在后续的lock之前发生，
从而满足了mutex的要求。
