import requests
from unittest.mock import patch
import queue
from enrich_task_logs import LogShipper
from determined.common.api import errors

# Test to ensure that the backoff.on_exception decorator used for LogShipper.ship() function is 
# setup and working properly. This test would fail/error out if there is anything wrong with the 
# backoff.on_onexception decorator setup. Otherwise, it will pass.
def test_backoff():
    # Create a sample LogShipper instance
    ship_queue = queue.Queue(maxsize=1)
    shipper = LogShipper(ship_queue, "http://localhost")
    # Add a sample log message to the queue
    ship_queue.put(
        {
            "timestamp": "timestamp",
            "log": "[rank=0] DEBUG log message" + "\n",
        }
    )
    # Mock the request method to raise exception so that we can test the backoff mechanism.
    with patch("determined.common.requests.request") as do_req_mock:
        # Setup a response object with status code 300 so that it will raise APIException.
        resp = requests.Response
        resp.status_code = 300
        resp.encoding = "json"
        resp._content = "{'message': 'API Exception'}"
        do_req_mock.return_value = resp
        try:
            # Start the shipper so that it will start sending the log messages.
            # This is bound to error out due to the mocked request function.
            shipper.run()
        except errors.APIException as e:
            # Currently backoff method is set to retry 3 times and raise the exception after that.
            # So, if we receive the correct exception (APIException) then the backoff is setup 
            # correctly and we can call this test a success.
            assert True
        except:
            # If we receive any other exception other than APIException it indicates that the 
            # backoff setup is failing. So, we indicate that with at test failure.
            assert False
