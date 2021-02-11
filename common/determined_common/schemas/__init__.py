# Avoid automatically importing any generated objects in this module, since those imports are
# non-trivial and would affect the user experience in the cli.
# TODO: rename schema.py to be _schema_base.py
from determined_common.schemas._auto_init import auto_init
from determined_common.schemas.schema import SchemaBase
from determined_common.schemas._union_base import UnionBase
