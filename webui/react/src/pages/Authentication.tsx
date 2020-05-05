import { Button, Form, Input } from 'antd';
import axios from 'axios';
import queryString from 'query-string';
import React, { useEffect, useState } from 'react';
import { useLocation } from 'react-router';
import { Redirect, useHistory } from 'react-router-dom';

import Icon from 'components/Icon';
import Logo, { LogoTypes } from 'components/Logo';
import Spinner from 'components/Spinner';
import Auth, { updateAuth } from 'contexts/Auth';
import handleError, { ErrorType } from 'ErrorHandler';
import { crossoverRoute, isCrossoverRoute } from 'routes';
import { isLoginFailure, login, logout } from 'services/api';
import { Credentials } from 'types';

import css from './Authentication.module.scss';

const DEFAULT_REDIRECT = '/det/dashboard';

interface Queries {
  redirect?: string;
}

interface FromValues {
  password?: string;
  username?: string;
}
const Authentication: React.FC = () => {
  const history = useHistory();
  const location = useLocation();
  const auth = Auth.useStateContext();
  const setAuth = Auth.useActionContext();
  const [ isLoading, setIsLoading ] = useState(true);
  const [ badCredentials, setBadCredentials ] = useState(false);
  const [ canSubmit, setCanSubmit ] = useState(false);

  const queries: Queries = queryString.parse(location.search);
  const redirectUri = queries.redirect || DEFAULT_REDIRECT;

  const isLogout = location.pathname.endsWith('logout');

  useEffect(() => {
    const source = axios.CancelToken.source();
    updateAuth(setAuth, source.token).then(() => setIsLoading(false));
    return (): void => {
      source.cancel();
    };
  }, [ setAuth ]);

  if (isLogout) {
    logout({});
    setAuth({ type: Auth.ActionType.Reset });
    history.push('/det/login' + location.search);
  }

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
      onValuesChange={onValuesChange}
    >
      <Form.Item
        name="username"
        rules={[
          {
            message: 'Please type in your username.',
            required: true,
          },
        ]}
      >
        <Input placeholder="Username" prefix={<Icon name="user-small" />} />
      </Form.Item>

      <Form.Item name="password">
        <Input.Password placeholder="Password" prefix={<Icon name="lock" />} />
      </Form.Item>

      {
        badCredentials &&
          <p className={[ css['error-message'], css.message ].join(' ')}>
            Incorrect username or password.
          </p>
      }

      <Form.Item>
        <Button disabled={!canSubmit} htmlType="submit" type="primary">
          Sign In
        </Button>
      </Form.Item>
    </Form>
  );

  if (auth.isAuthenticated) {
    if (isCrossoverRoute(redirectUri)) {
      crossoverRoute(redirectUri);
      return <Spinner fullPage />;
    }
    return <Redirect to={redirectUri} />;
  }
  if (isLogout || isLoading) return <Spinner fullPage />;

  return (
    <div className={css.base}>
      <div className={css.content}>
        <Logo className={css.logo} type={LogoTypes.Dark} />
        {loginForm}
        <p className={css.message}>
          Forgot your password, or need to manage users? Check out our
          <a href="/docs/system-administration/users.html?highlight=user"
            rel="noreferrer noopener" target="_blank">
            &nbsp;docs
          </a>
        </p>
      </div>
    </div>
  );
};

export default Authentication;
