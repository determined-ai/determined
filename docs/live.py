#!/usr/bin/env python3

import http.server
import os
import subprocess
import sys
import threading
import time

import watchdog
import watchdog.events
import watchdog.observers


def make_refresh_script(path, x_rebuild_time):
    """
    Create a snippet of javascript to inject into static pages.

    The snippet should reach back to our http server and long-poll for updates.
    """
    x = str(x_rebuild_time).encode("utf8")
    return (
        b'''
        <script>
            start_check = function() {
                let xhr = new XMLHttpRequest();
                xhr.open("POST", "/");
                xhr.setRequestHeader("X-rebuild-time", "'''
        + x
        + b'''");
                xhr.setRequestHeader("X-rebuild-path", "'''
        + path.encode("utf8")
        + b"""");
                xhr.send();
                xhr.onload = function() {
                    // alert(`Loaded: ${xhr.status} ${xhr.response}`);
                    if (xhr.response === "new") {
                        window.location.reload(false);
                    } else {
                        start_check();
                    }
                };
                xhr.onerror = function() {
                    // server is dead, refresh page to make that obvious.
                    window.location.reload(false);
                }
            }
            start_check()
        </script>
        """
    )


class LongPoller:
    """
    Pages which long-poll for updates come here to wait.

    Updates detected on the filesystem come here to be shared.
    """

    def __init__(self):
        self.cond = threading.Condition()
        self.should_continue = True
        self.rebuilds = {}

    def wait(self, path, x_rebuild_time):
        """
        Return True if the client should reload the page.
        """
        with self.cond:

            def out_of_date():
                if not self.should_continue:
                    # Never wait after quit().
                    return True
                return self.rebuilds.get(path, x_rebuild_time) != x_rebuild_time

            if not self.cond.wait_for(out_of_date, timeout=60):
                # Timeout.
                return False

            # Out-of-date.
            return True

    def quit(self):
        with self.cond:
            self.should_continue = False
            self.cond.notify_all()

    def update(self, path):
        try:
            stat = os.stat(path)
        except Exception as e:
            print(f"{path}: {e}")
            return
        with self.cond:
            self.rebuilds[path] = str(stat.st_mtime)
            # Alert the long-waiters.
            self.cond.notify_all()


def make_request_handler(long_poller, directory):
    class RequestHandler(http.server.SimpleHTTPRequestHandler):
        """
        RequestHandler is how we implement custom HTTP server behavior for GET and POST.
        """

        def __init__(self, *args):
            super().__init__(*args, directory=directory)

        def do_GET(self) -> None:
            """
            GET should inject snippets of javascript into static html pages.
            """
            path = self.translate_path(self.path)
            if path.startswith("site/html/") and os.path.isdir(path):
                path = os.path.join(path, "index.html")
            if path.endswith(".html"):
                # HTML documents get our client-side update script injected into them.
                try:
                    f = open(path, "rb")
                except OSError:
                    self.send_error(404, "file not found")
                    return None
                try:
                    stat = os.fstat(f.fileno())
                    body = f.read()
                finally:
                    f.close()

                # Inject the refresh script.
                body = body.replace(
                    b"<!--dev-reload-script-location-->", make_refresh_script(path, stat.st_mtime)
                )

                self.send_response(200)
                self.send_header("Content-Length", str(len(body)))
                self.send_header("Content-type", "text/html")
                self.send_header("Last-Modified", self.date_time_string(stat.st_mtime))
                self.end_headers()

                self.wfile.write(body)
                return None

            # Otherwise, use the default logic (mostly for downloads).
            return super().do_GET()

        def do_POST(self) -> None:
            """
            POST is exclusively for pages to long-poll for updates.
            """
            x_rebuild_time = self.headers["X-rebuild-time"]
            path = self.headers["X-rebuild-path"]
            assert x_rebuild_time and path
            new = long_poller.wait(path, x_rebuild_time)
            response = b"new" if new else b"old"
            try:
                self.send_response(200)
                self.send_header("Content-type", "application/text")
                self.send_header("Content-Length", str(len(response)))
                self.end_headers()
                self.wfile.write(response)
            except BrokenPipeError:
                # after waiting for so long, the client may have disconnected
                pass

    return RequestHandler


class Rebuilder(threading.Thread):
    """
    The Rebuilder thread calls the rebuild command whenever an input file has been updated.

    It ensures there are never two rebuild commands in flight.
    """

    def __init__(self):
        self.cond = threading.Condition()
        self.want_rebuild = False
        self.should_continue = True
        super().__init__()

    def run(self):
        while True:
            with self.cond:
                while self.should_continue and not self.want_rebuild:
                    self.cond.wait()
                if not self.should_continue:
                    return
                # Reset rebuild flag.
                self.want_rebuild = False
            # Rebuild.
            p = subprocess.Popen(["make", "build/sp-html.stamp"])
            ret = p.wait()
            if ret != 0:
                print("rebuild failed", file=sys.stderr)
            else:
                print("rebuild complete")

    def handle_input_update(self):
        with self.cond:
            self.want_rebuild = True
            self.cond.notify()

    def quit(self):
        with self.cond:
            self.should_continue = False
            self.cond.notify()


class FSHandler(watchdog.events.FileSystemEventHandler):
    """
    FSHandler detects file created and modified events.

    Updates to input files are sent to the Rebuilder.

    Updates to output files are sent to the LongPoller.
    """

    def __init__(self, rebuilder, long_poller):
        self.rebuilder = rebuilder
        self.long_poller = long_poller
        super().__init__()

    def on_created(self, event):
        self.handle(event)

    def on_modified(self, event):
        self.handle(event)

    def handle(self, event):
        if event.is_directory:
            return
        # Watchdog gives differently-styled src_paths on mac vs linux, so convert to relative path.
        path = os.path.relpath(event.src_path, ".")
        if path.startswith("site/"):
            if path.endswith(".html"):
                self.long_poller.update(path)
                return
        elif path.endswith(".rst") or path.startswith("assets/"):
            self.rebuilder.handle_input_update()


if __name__ == "__main__":
    long_poller = LongPoller()

    address = ("localhost", 1234)
    RequestHandler = make_request_handler(long_poller, directory="site/html/")
    server = http.server.ThreadingHTTPServer(address, RequestHandler)

    server_thread = threading.Thread(target=server.serve_forever, args=[0.1])
    server_thread.start()
    print(f"Listening on http://localhost:{address[1]}")

    rebuilder = Rebuilder()
    rebuilder.start()

    observer = watchdog.observers.Observer()

    observer.schedule(FSHandler(rebuilder, long_poller), ".", recursive=True)
    observer.start()

    try:
        while observer.is_alive():
            observer.join(1)
    finally:
        observer.stop()
        observer.join()
        rebuilder.quit()
        rebuilder.join()
        long_poller.quit()
        server.shutdown()
        server_thread.join()
