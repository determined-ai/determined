import { Button, Form, Input } from 'antd';
import React, { useCallback, useState } from 'react';

import Icon from 'components/Icon';
import Auth from 'contexts/Auth';
import FullPageSpinner from 'contexts/FullPageSpinner';
import handleError, { ErrorType } from 'ErrorHandler';
import { getCurrentUser, isLoginFailure, login } from 'services/api';
import { Credentials } from 'types';

import css from './DeterminedAuth.module.scss';

interface FromValues {
  password?: string;
  username?: string;
}

const DeterminedAuth: React.FC = () => {
  const setAuth = Auth.useActionContext();
  const setShowSpinner = FullPageSpinner.useActionContext();
  const [ isBadCredentials, setIsBadCredentials ] = useState(false);
  const [ canSubmit, setCanSubmit ] = useState(false);

  const onFinish = useCallback(async (creds: FromValues): Promise<void> => {
    setShowSpinner({ opaque: false, type: FullPageSpinner.ActionType.Show });
    setCanSubmit(false);
    try {
      await login(creds as Credentials);
      const user = await getCurrentUser({});
      setAuth({ type: Auth.ActionType.Set, value: { isAuthenticated: true, user } });
    } catch (e) {
      const actionMsg = isBadCredentials ? 'check your username and password.' : 'retry.';
      setShowSpinner({ type: FullPageSpinner.ActionType.Hide });
      setIsBadCredentials(isLoginFailure(e));
      handleError({
        error: e,
        isUserTriggered: true,
        message: e.message,
        publicMessage: `Failed to login. Please ${actionMsg}`,
        publicSubject: 'Login failed',
        silent: true,
        type: isBadCredentials ? ErrorType.Input : ErrorType.Server,
      });
    } finally {
      setCanSubmit(true);
    }
  }, [ isBadCredentials, setAuth, setShowSpinner ]);

  const onValuesChange = useCallback((changes: FromValues, values: FromValues): void => {
    const hasUsername = !!values.username;
    setIsBadCredentials(false);
    setCanSubmit(hasUsername);
  }, []);

  const loginForm = (
    <Form
      className={css.form}
      name="login"
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
        <Input autoFocus placeholder="username" prefix={<Icon name="user-small" size="small" />} />
      </Form.Item>
      <Form.Item name="password">
        <Input.Password placeholder="password" prefix={<Icon name="lock" size="small" />} />
      </Form.Item>
      {isBadCredentials && <p className={[ css.errorMessage, css.message ].join(' ')}>
        Incorrect username or password.
      </p>}
      <Form.Item>
        <Button disabled={!canSubmit} htmlType="submit" type="primary">Sign In</Button>
      </Form.Item>
    </Form>
  );

  return (
    <div className={css.base}>
      {loginForm}
      <p className={css.message}>
        Forgot your password, or need to manage users? Check out our&nbsp;
        <a href="/docs/topic-guides/users.html" rel="noreferrer noopener" target="_blank">docs</a>
      </p>
    </div>
  );

};

export default DeterminedAuth;
