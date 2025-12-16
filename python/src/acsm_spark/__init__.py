import grpc

from .manager_pb2 import ConnectSession, ConnectSessionRequest
from .manager_pb2_grpc import ACSManagerStub


class ACSManagerClient(ACSManagerStub):
    def __init__(self, addr: str):
        channel = grpc.insecure_channel(addr)
        super().__init__(channel)
