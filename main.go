package main

import(
	"time"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"sync"
	"runtime"
)

func data_init(_test_data []int) {
	for i := 0 ; i<len(_test_data); i++{
		_test_data[i] = 0
	}
}

func res_validate(_array []int, _stage_nb int) int{
	res := 0

	for  i := 0; i < len(_array); i++{
		if _array[i] != _stage_nb{
			res += 1
		}
	}

	return res
}

func add_serial(_num  int) int {
	_num += 1

	return _num
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
	_errnb = res_validate(_test_data, _stage_nb)
	fmt.Printf("Error rate %d/%d\n", _errnb, _len)
}

/*
 *	parallel part
*/

//chans init
func chans_init(_bfsz, _nb int)[]chan int{
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

	fmt.Println("[Force quit]", s)
}

//check goroutine
func res_recv(quit_chan chan int ,recv_chan chan int, reply_chan chan int, _recv_threshold int, _wg *sync.WaitGroup){
	_count := 0

	for true {
		select{
			//infinite loop untill signal
			case <-quit_chan:
				goto DONE

			//check last out chan
			case _data_in := <-recv_chan:
					//fmt.Println("Recv goroutine recv a data : ", _data_in)
					if _data_in >= 0 {
						_count += 1
					}
			default:
					//fmt.Println("Recv goroutine is runnning")
		}

		//reply master indicating all data process done
		if _count >= _recv_threshold{
			select{
				case reply_chan <- _count:
					fmt.Println("Recv goroutine : all data done")
					goto DONE
				default:
					goto DONE
			}
		}else{
			//fmt.Println("Recv goroutine : Wait for data, current count = ", _count)
		}

	}

DONE:
	fmt.Println("Recv goroutine quit")
	_wg.Done()

}

//parallel adder
func add_parallel(_test_data []int, c_in chan int, c_out chan int, c_quit chan int, stage_id int, _wg *sync.WaitGroup){
	//fmt.Println("Stage#", stage_id, "online")

	for true{
		select{
			case <-c_quit:
				goto DONE

			case _offset := <-c_in:
				//process data received and push to next stage
				//fmt.Println("Stage ", stage_id, " forward one data : ", _offset)
				_test_data[_offset] += 1
				c_out<- _offset

			default:
				//fmt.Println("Worker#", stage_id, " is running")
		}
	}

DONE:
	fmt.Println("worker#", stage_id, " stop...")
	_wg.Done()

	//fmt.Println("Stage#", stage_id, " quit")
}

//test for parallel process
func test_parallel(_test_data []int, _tot_nb, _stage_nb int){
	//init all chans
	_data_chans := chans_init(2, _stage_nb + 1)
	_quit_chans := chans_init(1, _stage_nb + 2)
	_count := 0
	var wg sync.WaitGroup

	////monitor system signal
	//go handle_signal(_quit_chans)

	wg.Add(_stage_nb + 1)

	for i := 0 ; i < _stage_nb ; i++{
		go add_parallel(_test_data, _data_chans[i], _data_chans[i+1], _quit_chans[i], i, &wg)
	}

	//start the goroutine to receive 
	_res_chan := make(chan int, 2)
	go res_recv(_quit_chans[_stage_nb], _data_chans[_stage_nb], _res_chan, _tot_nb, &wg)

	time.Sleep(1)

	_t1 := time.Now()

	//push all data to channel one by one
	for i := 0; i<_tot_nb; {
		select{
			case <-_quit_chans[_stage_nb + 1]:
				goto FOR_QUIT

			case _data_chans[0] <- i:
				//fmt.Println("Push data#", i)
				i++

			default:
		}
	}

	//receive res from res_check goroutine
	DONE2:
	for true{
		select{
			//global force quit
			case <-_quit_chans[_stage_nb + 1]:
				//fmt.Println("Main force quit")
				break DONE2

			//poll recv goroutine
			case _res_nb, _res_ok := <-_res_chan:
				if _res_ok == true {
					//fmt.Println("Main normal quit")
					_count += _res_nb
					break DONE2
				}

			default:
		}
	}

FOR_QUIT:
	_t2 := time.Since(_t1)
	fmt.Println("Tot ", _count)
	fmt.Println("Time elapsed ", _t2)

	for i := 0; i < len(_quit_chans); i++{
		_quit_chans[i] <- 1
	}

	fmt.Println("Try to stop all goroutine")

	wg.Wait()


	//close all chans
	for i :=0; i < (_stage_nb + 1); i++{
		close(_data_chans[i])
	}

	for i :=0; i < (_stage_nb + 2); i++{
		close(_quit_chans[i])
	}

	close(_res_chan)
}

func main(){
	const SIZE int  = (1 << 12)
	var stage_nb int = 8

	//set max cpu number can be involved
	runtime.GOMAXPROCS(8)

	//data slice
	var test_array []int = make([]int, SIZE)

	//init data
	data_init(test_array)

	//serial test
	fmt.Println("Serial test:")
	test_serial(test_array, stage_nb)

	//parallel test
	fmt.Println("Parallel test:")
	test_parallel(test_array, len(test_array), stage_nb)
}
