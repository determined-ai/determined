:orphan:

**Breaking Changes**

-  Python SDK and CLI: Enforce password requirements for all non-remote users, aligning with WebUI
   password standards and having the following requirements:

      -  Passwords must be at least 8 characters long (and not None).

      -  Passwords must contain at least one upper-case letter.

      -  Passwords must contain at least one lower-case letter.

      -  Passwords must contain at least one number.

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
