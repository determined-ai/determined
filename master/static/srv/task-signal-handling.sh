#!/usr/bin/env bash

trap_and_forward_signals() {
	handle_signals() {
		sig="$1"
		shift
		trapped_signal="yes"
		if [ "${wait_child_pid}" ]; then
			# If the child process isn't alive yet, then this is OK, whoever can just resend the signal.
			kill -s "$sig" "${wait_child_pid}" 2>/dev/null
		fi
	}

	trap_and_capture_signal() {
		func="$1"
		shift
		for sig in "$@"; do
			trap "$func $sig" "$sig"
		done
	}

	unset wait_child_pid
	unset trapped_signal
	trap_and_capture_signal 'handle_signals' TERM INT SIGUSR1 SIGUSR2
}

wait_and_handle_signals() {
	wait_child_pid=$1

	while true; do
		set +e
		wait $wait_child_pid
		wait_child_exit=$?
		set -e

		# When a signal is sent to the shell, it will interrupt waits, after all traps have run. To
		# discern if the wait unblocked because of a signal or process exit, we set "trapped_signal"
		# in traps and check it here.
		if [ -z "${trapped_signal}" ]; then
			exit $wait_child_exit
		else
			unset trapped_signal
		fi
	done
}
