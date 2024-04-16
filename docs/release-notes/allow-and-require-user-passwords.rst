:orphan:

**Breaking Changes**

-  Python SDK and CLI: Enforce password requirements for all non-remote users: see
   :ref:`password-requirements`

      -  This applies to ``create_user``, ``User.change_password``, ``det user create``, and ``det
         user change-password``.

      -  This does not affect logging in with an existing user who already has an empty or
         non-compliant password, but we recommend setting good passwords for such users as soon as
         possible.

-  CLI: Require and allow passwords to be set when creating non-remote users with ``det user
   create``.

      -  This may be done interactively by following the prompts.
      -  This may be done noninteractively by using the ``--password`` option.
      -  This is not required when creating a user with ``--remote`` since Single Sign-On will be
         used.
