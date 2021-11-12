import determined as det
from determined.experimental import client
import b

if __name__ == "__main__":
    client.login()
    session = client._determined._session
    print(b.get_Determined_GetMasterConfig(session).to_json())
