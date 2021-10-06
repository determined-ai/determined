import sys
import zmq
import json

context = zmq.Context()
socket = context.socket(zmq.REQ)
socket.connect('tcp://127.0.0.1:5555')
args = sys.argv[1:]
# socket.send(pickle.dumps(args))
socket.send_string(json.dumps(args))
# msg = socket.recv()
# print(msg)
