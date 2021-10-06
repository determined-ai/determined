#!/usr/bin/env python
import json
from determined.cli.cli import main
import argparse
import zmq

parser = argparse.ArgumentParser(description='zeromq server/client')
parser.add_argument('--bar')
args = parser.parse_args()

# server
context = zmq.Context()
socket = context.socket(zmq.REP)
socket.bind('tcp://127.0.0.1:5555')
while True:
    msg = socket.recv()
    msg = json.loads(msg)
    print(msg)
    try:
        main(msg)
    except Exception as e:
        print(e)
    if msg == 'zeromq':
        socket.send_string('ah ha!')
    else:
        socket.send_string('...nah')
