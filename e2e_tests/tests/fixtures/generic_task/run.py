import time

t = 5
print("start")
for i in range(t):
    if i % 10 == 0:
        print("Working...")
    time.sleep(1)
print("end")
