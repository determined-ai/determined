import requests

from ..models.base import base_t


class DetAPIClient:
    def __init__(self, det_master: str, username: str, password: str):
        response = requests.post(f"{det_master}/api/v1/auth/login",
                                 json={"username": username, "password": password})
        if not response.ok:
            raise Exception("Failed to login")
        self.token = response.json()['token']
        self.det_master = det_master
    
    def get(self, url: str) -> base_t:
        response = requests.get(f"{self.det_master}{url}",
                                headers={"Authorization": f"Bearer {self.token}"})
        return response.json()

    def logout(self):
        requests.post(f"{self.det_master}/api/v1/auth/login",
                      headers={"Authorization": f"Bearer {self.token}"})
