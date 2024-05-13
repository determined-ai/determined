// TODO: DET-10209 - obtain these values from an endpoint

const NEW_PASSWORD_REQUIRED_MESSAGE = "Password can't be blank";
const PASSWORD_TOO_SHORT_MESSAGE = 'Password must have at least 8 characters';
const PASSWORD_UPPERCASE_MESSAGE = 'Password must include an uppercase letter';
const PASSWORD_LOWERCASE_MESSAGE = 'Password must include a lowercase letter';
const PASSWORD_NUMBER_MESSAGE = 'Password must include a number';

export const PASSWORD_RULES = [
  { message: NEW_PASSWORD_REQUIRED_MESSAGE, required: true },
  { message: PASSWORD_TOO_SHORT_MESSAGE, min: 8 },
  {
    message: PASSWORD_UPPERCASE_MESSAGE,
    pattern: /[A-Z]+/,
  },
  {
    message: PASSWORD_LOWERCASE_MESSAGE,
    pattern: /[a-z]+/,
  },
  {
    message: PASSWORD_NUMBER_MESSAGE,
    pattern: /\d/,
  },
];
