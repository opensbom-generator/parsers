# SPDX-License-Identifier: Apache-2.0
from fastapi import FastAPI
from cryptography.fernet import Fernet
app = FastAPI()


@app.get("/")
def key_generator():
    key = Fernet.generate_key()
    return {"key": key}
