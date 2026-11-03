"""A least-recently-used cache: when full, the entry that was read longest ago is dropped."""
from collections import OrderedDict


class LRUCache:
    def __init__(self, capacity: int) -> None:
        self.capacity = capacity
        self._items: OrderedDict[str, object] = OrderedDict()

    def get(self, key: str):
        if key not in self._items:
            return None
        self._items.move_to_end(key)
        return self._items[key]

    def put(self, key: str, value: object) -> None:
        self._items[key] = value
        self._items.move_to_end(key)
        if len(self._items) > self.capacity:
            self._items.popitem(last=False)
