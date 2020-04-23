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
import { login, logout } from 'services/api';
import { Credentials } from 'types';

import css from './Authentication.module.scss';

// TODO support custom rediret param
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
    console.log('is in logout page');
    logout({});
    setAuth({ type: Auth.ActionType.Reset });
    history.push('/det/login' + props.location.search);
  } else {
    console.log('is in login page');
  }

  const onFinish = (creds: unknown): void => {
    // TODO validate the creds type?
    const hideLoader = message.loading('logging in..');
    login(creds as Credentials)
      .then(() => {
        setAuth({ type: Auth.ActionType.SetIsAuthenticated, value: true });
      })
      .catch((e: Error) => {
        // TODO check for the code or error type?
        handleError({
          error: e,
          isUserTriggered: true,
          message: e.message,
          publicMessage: 'Failed to login. Please check your username and password.',
          publicSubject: 'Login failed',
          type: ErrorType.Input,
        });
      })
      .finally(hideLoader);
  };

  const loginForm = (
    <Form
      name="basic"
      size="large"
      onFinish={onFinish}
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

      <Form.Item>
        <Button htmlType="submit" type="primary">
          Sign In
        </Button>
      </Form.Item>
    </Form>

  );

  if (isLogout || isLoading) return <Spinner fullPage />;
  if (auth.isAuthenticated) return <Redirect to={redirectUri} />;

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
