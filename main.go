package main

import(
	"time"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func data_init(_test_data []int) {
	for i := 0 ; i<len(_test_data); i++{
		_test_data[i] = 0
	}
}

func res_validate(_array []int) int{
	res := 0

	for  i := 0; i < len(_array); i++{
		if _array[i] != 3{
			res += 1
		}
	}

	return res
}

func add_serial(_num  int) int {
	_num += 1

	return _num
}

func add_parallel(c_in chan *int, c_out chan *int, c_quit chan int){
	
	for true{
		_sig, _res := <-c_quit
		if _res == true{
			fmt.Println(_sig)
			break
		}

		_num, _ok := <-c_in

		if _ok == true {
			*_num += 1
			c_out <- _num
		}
	}
}

//test for serial process
func test_serial(_test_data []int, _stage_nb int){
	//time elapsed measure
	t1 := time.Now()
	_len := len(_test_data)

	//serial add
	for i := 0; i < _stage_nb; i++{
		for j := 0 ; j < _len; j++{
			_test_data[j] = add_serial( _test_data[j] )
		}
	}

	t2 := time.Since(t1)

	fmt.Println("Time elapsed ", t2)

	//result validate
	var _errnb int
	_errnb = res_validate(_test_data)
	fmt.Printf("Error rate %d/%d\n", _errnb, _len)
}

//chans init
func chans_init(_bfsz, _nb int)[]chan *int{
	_chans := make([]chan *int, _nb)
	for i := range _chans{
		_chans[i] = make(chan *int, _bfsz)
	}

	return _chans
}

//quit chans init
func quit_chans_init(_bfsz, _nb int)[]chan int{
	_chans := make([]chan int, _nb)
	for i := range _chans{
		_chans[i] = make(chan int, _bfsz)
	}

	return _chans
}

//signal for ctrl+c
func handle_signal(_quit_chans []chan int){
	sys_chan := make(chan os.Signal)

	signal.Notify(sys_chan, os.Interrupt, os.Kill, syscall.SIGUSR1, syscall.SIGUSR2)

	s := <-sys_chan

	for i := 0; i < len(_quit_chans); i++{
		_quit_chans[i] <- 1
	}

	fmt.Println("Force quit ", s)
}

//test for parallel process
func test_parallel(_test_data []int, _stage_nb int){
	//keep all goroutine alive
	_count := 0
	
	//init all chans
	_in_chans := chans_init(10, _stage_nb)
	_out_chans := chans_init(10, _stage_nb)
	_quit_chans := quit_chans_init(1, _stage_nb + 1)
	
	//monitor system signal
	go handle_signal(_quit_chans, )

	for i := 0 ; i < _stage_nb ; i++{
		go add_parallel(_in_chans[i], _out_chans[i], _quit_chans[i])
	}

	var _res *int
	_ok := true
	for true {
		_quit, _quit_res := <-_quit_chans[_stage_nb]
		if _quit_res == true{
			fmt.Println("Main:", _quit)
			break
		}

		_res, _ok = <-_out_chans[_stage_nb - 1]
		if _ok == true{
			_count += 1
		}

		fmt.Println(*_res)
	}

}

func main(){
	const SIZE int  = (1 << 21)
	var stage_nb int = 3

	//data slice
	var test_array []int = make([]int, SIZE)

	//init data
	data_init(test_array)

	//serial test
	test_serial(test_array, stage_nb)

	//parallel test
	test_parallel(test_array, stage_nb)
}
