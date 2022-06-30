#!/bin/zsh  

for i in 1 2 3 4 5 6 7 8 9 10 11 12
do
  touch "tempor$i"
  git add . && git commit -m "test-$1"
  sleep 300000
done

