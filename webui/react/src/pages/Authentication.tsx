import { Button, Form, Input } from 'antd';
import { message } from 'antd';
import axios from 'axios';
import queryString from 'query-string';
import React, { useEffect, useState } from 'react';
import { useHistory, useLocation } from 'react-router';
import { Redirect } from 'react-router-dom';

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

type WithSearch<T> = T & {location: {search: string}};
interface Queries {
  redirect?: string;
}

const Authentication: React.FC<WithSearch<{}>> = (props: WithSearch<{}>) => {
  const history = useHistory();
  const location = useLocation();
  const auth = Auth.useStateContext();
  const setAuth = Auth.useActionContext();
  const [ isLoading, setIsLoading ] = useState(true);
  const [ badCredentials, setBadCredentials ] = useState(false);
  const [ canSubmit, setCanSubmit ] = useState(false);

  const queries: Queries = queryString.parse(props.location.search);

  const redirectUri= queries.redirect || DEFAULT_REDIRECT;

  useEffect(() => {
    const source = axios.CancelToken.source();
    updateAuth(setAuth, source.token).then(() => setIsLoading(false));
    return (): void => {
      source.cancel();
    };
  }, [ setAuth ]);

  const isLogout = location.pathname.endsWith('logout');
  if (isLogout) {
    logout({});
    setAuth({ type: Auth.ActionType.Reset });
    history.push('/det/login' + props.location.search);
  }

  const onFinish = (creds: unknown): void => {
    // TODO validate the creds type?
    setCanSubmit(false);
    const hideLoader = message.loading('logging in..');
    login(creds as Credentials)
      .then(() => updateAuth(setAuth))
      .catch((e: Error) => {
        setBadCredentials(isLoginFailure(e));
        if (badCredentials) return;
        // DISCUSS do we want to potentially report it?
        // or pass it through error handler for some reason?
        const actionMsg = badCredentials ? 'check your username and password.' : 'retry.';
        handleError({
          error: e,
          isUserTriggered: true,
          message: e.message,
          publicMessage: `Failed to login. Please ${actionMsg}`,
          publicSubject: 'Login failed',
          type: badCredentials ? ErrorType.Input : ErrorType.Server,
        });
      })
      .finally(() => {
        setCanSubmit(true);
        hideLoader();
      });
  };

  const onValuesChange = (): void => {
    setBadCredentials(false);
    setCanSubmit(true);
  };

  const loginForm = (
    <Form
      name="basic"
      size="large"
      onFinish={onFinish}
      onValuesChange={onValuesChange}
    >
      <Form.Item
        name="username"
        rules={[
          {
            message: 'Please input your username!',
            required: true,
          },
        ]}
      >
        <Input placeholder="Username" prefix={<Icon name="user" />} />
      </Form.Item>

      <Form.Item name="password">
        <Input.Password placeholder="Password" prefix={<Icon name="lock" />} />
      </Form.Item>

      {
        badCredentials &&
          <p className={css['error-message']}>
            Incorrect username or password, please try again.
          </p>
      }

      <Form.Item>
        <Button disabled={!canSubmit} htmlType="submit" type="primary">
          Sign In
        </Button>
      </Form.Item>
    </Form>

  );

  if (isLogout || isLoading) return <Spinner fullPage />;
  if (auth.isAuthenticated) {
    if (isCrossoverRoute(redirectUri)) {
      crossoverRoute(redirectUri);
      return <Spinner fullPage />;
    }
    return <Redirect to={redirectUri} />;
  }

  return (
    <div className={css.base}>
      <div className={css.content}>
        {/* DISCUSSION what if we didn't need to add the logo classname and was able to
        target logo on its own using component name easily in module.scss */}
        <Logo className={css.logo} type={LogoTypes.Dark} />
        {loginForm}
        <a href="/docs/system-administration/users.html?highlight=user" target="_blank">
          Forgot your password or need to create a user? Checkout our docs
        </a>
      </div>
    </div>
  );
};

export default Authentication;
