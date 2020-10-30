import { Button, Form, Input } from 'antd';
import React, { useCallback, useState } from 'react';

import Icon from 'components/Icon';
import Link from 'components/Link';
import Auth from 'contexts/Auth';
import UI from 'contexts/UI';
import handleError, { ErrorType } from 'ErrorHandler';
import { getCurrentUser, isLoginFailure, login } from 'services/api';
import { updateDetApi } from 'services/apiConfig';
import { Credentials } from 'types';
import { Storage } from 'utils/storage';

import css from './DeterminedAuth.module.scss';

interface FromValues {
  password?: string;
  username?: string;
}

const storage = new Storage({ basePath: '/DeterminedAuth', store: window.localStorage });
const STORAGE_KEY_LAST_USERNAME = 'lastUsername';

const DeterminedAuth: React.FC = () => {
  const setAuth = Auth.useActionContext();
  const setUI = UI.useActionContext();
  const [ isBadCredentials, setIsBadCredentials ] = useState(false);
  const [ canSubmit, setCanSubmit ] = useState(!!storage.get(STORAGE_KEY_LAST_USERNAME));

  const onFinish = useCallback(async (creds: FromValues): Promise<void> => {
    setUI({ type: UI.ActionType.ShowSpinner });
    setCanSubmit(false);
    try {
      const { token } = await login(creds as Credentials);
      updateDetApi({ apiKey: 'Bearer ' + token });
      const user = await getCurrentUser({});
      setAuth({ type: Auth.ActionType.Set, value: { isAuthenticated: true, token, user } });
      storage.set(STORAGE_KEY_LAST_USERNAME, creds.username);
    } catch (e) {
      const isBadCredentialsSync = isLoginFailure(e);
      setIsBadCredentials(isBadCredentialsSync); // this is not a sync operation
      setUI({ type: UI.ActionType.HideSpinner });
      const actionMsg = isBadCredentialsSync ? 'check your username and password.' : 'retry.';
      if (isBadCredentialsSync) storage.remove(STORAGE_KEY_LAST_USERNAME);
      handleError({
        error: e,
        isUserTriggered: true,
        message: e.message,
        publicMessage: `Failed to login. Please ${actionMsg}`,
        publicSubject: 'Login failed',
        silent: true,
        type: isBadCredentialsSync ? ErrorType.Input : ErrorType.Server,
      });
    } finally {
      setCanSubmit(true);
    }
  }, [ setAuth, setUI ]);

  const onValuesChange = useCallback((changes: FromValues, values: FromValues): void => {
    const hasUsername = !!values.username;
    setIsBadCredentials(false);
    setCanSubmit(hasUsername);
  }, []);

  const loginForm = (
    <Form
      className={css.form}
      initialValues={{ username: storage.getWithDefault(STORAGE_KEY_LAST_USERNAME, '') }}
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
        <Link path={'/docs/topic-guides/users.html'} popout>docs</Link>
      </p>
    </div>
  );
};

export default DeterminedAuth;
