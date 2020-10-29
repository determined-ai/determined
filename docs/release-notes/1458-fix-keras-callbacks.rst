:orphan:

**Fixes**

- Fix support for Keras Callbacks.

  - Previously, stateful Keras Callbacks (``EarlyStopping`` and
    ``ReduceLROnPlateau``) did not work in Determined across pause/activate
    boundaries.  We have introduced Determined-friendly implementations,
    :class:`determined.keras.callbacks.EarlyStopping` and
    :class:`determined.keras.callbacks.ReduceLROnPlateau`, which address this
    shortcoming.  User-defined callbacks may subclass
    :class:`determined.keras.callbacks.Callback` (and define ``get_state`` and
    ``load_state`` methods) to also benfit from this and other new features.

  - Previously, Keras Callbacks which relied on ``on_epoch_end`` in Determined
    would see their ``on_epoch_end`` called every ``scheduling_unit`` batches
    by default.  Now, ``on_epoch_end`` will be reliably called at the end of
    each epoch, as defined by the ``records_per_epoch`` setting in the
    experiment config.  As before, ``on_epoch_end`` will not contain validation
    metrics, as the validation data is not always fresh at epoch boundaries.
    Therefore, the Determined implementations of
    :class:`~determined.keras.callbacks.EarlyStopping` and
    :class:`~determined.keras.callbacks.ReduceLROnPlateau` are both based on
    ``on_test_end``, which can be tuned using ``min_validation_period``.
