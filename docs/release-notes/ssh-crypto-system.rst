:orphan:

**Improvements**

-  Master Configuration: Add support for crypto system configuration for ssh connection.
   ``security.key_type`` now accepts ``RSA``, ``ECDSA`` or ``ED25519``. Default key type is changed
   from ``1024-bit RSA`` to ``ED25519``, since ``ED25519`` keys are faster and more secure than the
   old default, and ``ED25519`` is also the default key type for ``ssh-keygen``.
