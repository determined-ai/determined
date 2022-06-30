#!/bin/sh  
while true  
:Q
for i in 1 2 3 4 5 6 7 8 9 10 11 12
do
  touch "tempor$i"
  gac 'no-op'
  sleep 300000
done

