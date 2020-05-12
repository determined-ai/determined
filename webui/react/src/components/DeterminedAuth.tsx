import { Button, Form, Input } from 'antd';
import React, { useState } from 'react';

import Icon from 'components/Icon';
import Auth, { updateAuth } from 'contexts/Auth';
import handleError, { ErrorType } from 'ErrorHandler';
import { isLoginFailure, login } from 'services/api';
import { Credentials } from 'types';

import css from './DeterminedAuth.module.scss';

interface FromValues {
  password?: string;
  username?: string;
}

interface Props {
  setIsLoading: (isLoading: boolean) => void;
}

const DeterminedAuth: React.FC<Props> = ({ setIsLoading }: Props) => {
  const setAuth = Auth.useActionContext();
  const [ badCredentials, setBadCredentials ] = useState(false);
  const [ canSubmit, setCanSubmit ] = useState(false);

  const onFinish = async (creds: FromValues): Promise<void> => {
    setIsLoading(true);
    setCanSubmit(false);
    try {
      await login(creds as Credentials);
      updateAuth(setAuth);
    } catch (e) {
      setIsLoading(false);
      setBadCredentials(isLoginFailure(e));
      const actionMsg = badCredentials ? 'check your username and password.' : 'retry.';
      handleError({
        error: e,
        isUserTriggered: true,
        message: e.message,
        publicMessage: `Failed to login. Please ${actionMsg}`,
        publicSubject: 'Login failed',
        silent: true,
        type: badCredentials ? ErrorType.Input : ErrorType.Server,
      });
    } finally {
      setCanSubmit(true);
    }
  };

  const onValuesChange = (changes: FromValues, values: FromValues): void => {
    const hasUsername = !!values.username;
    setBadCredentials(false);
    setCanSubmit(hasUsername);
  };

  const loginForm = (
    <Form
      name="login"
      size="large"
      onFinish={onFinish}
      onValuesChange={onValuesChange}>
      <Form.Item
        name="username"
        rules={[
          {
            message: 'Please type in your username.',
            required: true,
          },
        ]}>
        <Input placeholder="Username" prefix={<Icon name="user-small" />} />
      </Form.Item>

      <Form.Item name="password">
        <Input.Password placeholder="Password" prefix={<Icon name="lock" />} />
      </Form.Item>

      {badCredentials && <p className={[ css.errorMessage, css.message ].join(' ')}>
            Incorrect username or password.
      </p>}

      <Form.Item>
        <Button disabled={!canSubmit} htmlType="submit" type="primary">
          Sign In
        </Button>
      </Form.Item>
    </Form>
  );

  return (
    <div className={css.base}>
      {loginForm}
      <p className={css.message}>
          Forgot your password, or need to manage users? Check out our
        <a href="/docs/system-administration/users.html?highlight=user"
          rel="noreferrer noopener" target="_blank">
            &nbsp;docs
        </a>
      </p>
    </div>
  );

};

export default DeterminedAuth;
