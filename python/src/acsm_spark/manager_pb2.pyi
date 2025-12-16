from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ConnectSessionRequest(_message.Message):
    __slots__ = ("session_name",)
    SESSION_NAME_FIELD_NUMBER: _ClassVar[int]
    session_name: str
    def __init__(self, session_name: _Optional[str] = ...) -> None: ...

class ConnectSession(_message.Message):
    __slots__ = ("session_id", "session_name")
    SESSION_ID_FIELD_NUMBER: _ClassVar[int]
    SESSION_NAME_FIELD_NUMBER: _ClassVar[int]
    session_id: str
    session_name: str
    def __init__(self, session_id: _Optional[str] = ..., session_name: _Optional[str] = ...) -> None: ...

class ConnectSessionListRequest(_message.Message):
    __slots__ = ("page_number", "page_size")
    PAGE_NUMBER_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    page_number: int
    page_size: int
    def __init__(self, page_number: _Optional[int] = ..., page_size: _Optional[int] = ...) -> None: ...

class ConnectSessionListResponse(_message.Message):
    __slots__ = ("sessions", "total_sessions", "page_number", "page_size")
    SESSIONS_FIELD_NUMBER: _ClassVar[int]
    TOTAL_SESSIONS_FIELD_NUMBER: _ClassVar[int]
    PAGE_NUMBER_FIELD_NUMBER: _ClassVar[int]
    PAGE_SIZE_FIELD_NUMBER: _ClassVar[int]
    sessions: _containers.RepeatedCompositeFieldContainer[ConnectSession]
    total_sessions: int
    page_number: int
    page_size: int
    def __init__(self, sessions: _Optional[_Iterable[_Union[ConnectSession, _Mapping]]] = ..., total_sessions: _Optional[int] = ..., page_number: _Optional[int] = ..., page_size: _Optional[int] = ...) -> None: ...
