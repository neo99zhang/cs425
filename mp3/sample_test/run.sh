#!/bin/bash

code_folder='src/'
curr_folder=$(pwd)'/'
servers=(A B C D E)
server_pids=()

# shutdown servers
shutdown() {
	for pid in ${server_pids[@]}; do
		kill -9 $pid
	done
}
trap shutdown EXIT

# initialize servers
cp config.txt ${code_folder}/config.txt
cd $code_folder
for server in ${servers[@]}; do
	./server $server config.txt > ../server_${server}.log 2>&1 &
	server_pids+=($!)
done

# run 2 tests
timeout 5s ./client a config.txt < ${curr_folder}input1.txt > ${curr_folder}output1.log 2>&1
timeout 5s ./client a config.txt < ${curr_folder}input2.txt > ${curr_folder}output2.log 2>&1

cd $curr_folder
echo "Difference between your output and expected output:"
diff output1.log expected1.txt
diff output2.log expected2.txt

shutdown