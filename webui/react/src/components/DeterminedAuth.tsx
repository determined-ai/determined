import { Button, Form, Input } from 'antd';
import React, { useCallback, useState } from 'react';

import Link from 'components/Link';
import { StoreAction, useStoreDispatch } from 'contexts/Store';
import { paths } from 'routes/utils';
import { login } from 'services/api';
import { updateDetApi } from 'services/apiConfig';
import { isLoginFailure } from 'services/utils';
import Icon from 'shared/components/Icon/Icon';
import { ErrorType } from 'shared/utils/error';
import { Storage } from 'shared/utils/storage';
import handleError from 'utils/error';

import css from './DeterminedAuth.module.scss';

interface Props {
  canceler: AbortController;
}

interface FromValues {
  password?: string;
  username?: string;
}

const storage = new Storage({ basePath: '/DeterminedAuth', store: window.localStorage });
const STORAGE_KEY_LAST_USERNAME = 'lastUsername';

const DeterminedAuth: React.FC<Props> = ({ canceler }: Props) => {
  const storeDispatch = useStoreDispatch();
  const [ isBadCredentials, setIsBadCredentials ] = useState(false);
  const [ canSubmit, setCanSubmit ] = useState(!!storage.get(STORAGE_KEY_LAST_USERNAME));

  const onFinish = useCallback(async (creds: FromValues): Promise<void> => {
    storeDispatch({ type: StoreAction.ShowUISpinner });
    setCanSubmit(false);
    try {
      const { token, user } = await login(
        {
          password: creds.password || '',
          username: creds.username || '',
        }
        , { signal: canceler.signal },
      );
      updateDetApi({ apiKey: `Bearer ${token}` });
      storeDispatch({
        type: StoreAction.SetAuth,
        value: { isAuthenticated: true, token, user },
      });
      storage.set(STORAGE_KEY_LAST_USERNAME, creds.username);
    } catch (e) {
      const isBadCredentialsSync = isLoginFailure(e);
      setIsBadCredentials(isBadCredentialsSync); // this is not a sync operation
      storeDispatch({ type: StoreAction.HideUISpinner });
      const actionMsg = isBadCredentialsSync ? 'check your username and password.' : 'retry.';
      if (isBadCredentialsSync) storage.remove(STORAGE_KEY_LAST_USERNAME);
      handleError(e, {
        isUserTriggered: true,
        publicMessage: `Failed to login. Please ${actionMsg}`,
        publicSubject: 'Login failed',
        silent: false,
        type: isBadCredentialsSync ? ErrorType.Input : ErrorType.Server,
      });
    } finally {
      setCanSubmit(true);
    }
  }, [ canceler, storeDispatch ]);

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
      {isBadCredentials && (
        <p className={[ css.errorMessage, css.message ].join(' ')}>
          Incorrect username or password.
        </p>
      )}
      <Form.Item>
        <Button
          disabled={!canSubmit}
          htmlType="submit"
          loading={!canSubmit}
          type="primary">
          Sign In
        </Button>
      </Form.Item>
    </Form>
  );

  return (
    <div className={css.base}>
      {loginForm}
      <p className={css.message}>
        Forgot your password, or need to manage users? Check out our&nbsp;
        <Link external path={paths.docs('/sysadmin-basics/users.html')} popout>docs</Link>
      </p>
    </div>
  );
};

export default DeterminedAuth;
