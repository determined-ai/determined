import { CheckOutlined, CloseOutlined, EditOutlined } from '@ant-design/icons';
import { Divider, Typography } from 'antd';
import React, { useCallback, useState } from 'react';

import { Size } from 'components/Avatar';
import Button from 'components/kit/Button';
import Form from 'components/kit/Form';
import Input from 'components/kit/Input';
import { useModal } from 'components/kit/Modal';
import Avatar from 'components/kit/UserAvatar';
import PasswordChangeModalComponent from 'components/PasswordChangeModal';
import { patchUser } from 'services/api';
import determinedStore from 'stores/determinedInfo';
import userStore from 'stores/users';
import { message } from 'utils/dialogApi';
import { ErrorType } from 'utils/error';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';
import { useObservable } from 'utils/observable';

import css from './SettingsAccount.module.scss';

interface FormUsernameInputs {
  username: string;
}

interface FormDisplaynameInputs {
  displayName: string;
}

export const API_DISPLAYNAME_SUCCESS_MESSAGE = 'Display name updated.';
export const API_USERNAME_ERROR_MESSAGE = 'Could not update username.';
export const API_USERNAME_SUCCESS_MESSAGE = 'Username updated.';
export const CHANGE_PASSWORD_TEXT = 'Change Password';

const SettingsAccount: React.FC = () => {
  const [usernameForm] = Form.useForm<FormUsernameInputs>();
  const [displaynameForm] = Form.useForm<FormDisplaynameInputs>();
  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));
  const [isUsernameEditable, setIsUsernameEditable] = useState<boolean>(false);
  const [isDisplaynameEditable, setIsDisplaynameEditable] = useState<boolean>(false);
  const info = useObservable(determinedStore.info);

  const PasswordChangeModal = useModal(PasswordChangeModalComponent);

  const handleSaveDisplayName = useCallback(async (): Promise<void | Error> => {
    const values = await displaynameForm.validateFields();
    try {
      const user = await patchUser({
        userId: currentUser?.id || 0,
        userParams: { displayName: values.displayName },
      });
      userStore.updateUsers(user);
      message.success(API_DISPLAYNAME_SUCCESS_MESSAGE);
      setIsDisplaynameEditable(false);
    } catch (e) {
      handleError(e, { silent: false, type: ErrorType.Input });
      return e as Error;
    }
  }, [currentUser?.id, displaynameForm]);

  const handleSaveUsername = useCallback(async (): Promise<void | Error> => {
    const values = await usernameForm.validateFields();
    try {
      const user = await patchUser({
        userId: currentUser?.id || 0,
        userParams: { username: values.username },
      });
      userStore.updateUsers(user);
      message.success(API_USERNAME_SUCCESS_MESSAGE);
      setIsUsernameEditable(false);
    } catch (e) {
      message.error(API_USERNAME_ERROR_MESSAGE);
      handleError(e, { silent: true, type: ErrorType.Input });
      return e as Error;
    }
  }, [currentUser?.id, usernameForm]);

  return (
    <div className={css.base}>
      <div className={css.avatar}>
        <Avatar hideTooltip size={Size.ExtraLarge} user={currentUser} />
      </div>
      <Divider />
      <div className={css.row}>
        <label>Username</label>
        {!isUsernameEditable ? (
          <div className={css.displayInfo}>
            <span>{currentUser?.username ?? ''}</span>
            <Button
              data-testid="edit-username"
              disabled={!info.userManagementEnabled}
              icon={<EditOutlined />}
              onClick={() => setIsUsernameEditable(true)}
            />
          </div>
        ) : (
          <Form
            className={css.form}
            form={usernameForm}
            layout="inline"
            onFinish={handleSaveUsername}>
            <Form.Item
              initialValue={currentUser?.username ?? ''}
              name="username"
              noStyle
              rules={[{ message: 'Please input your username', required: true }]}>
              <Input maxLength={32} placeholder="Add username" style={{ widows: '80%' }} />
            </Form.Item>
            <Form.Item noStyle>
              <Button
                disabled={!info.userManagementEnabled}
                htmlType="submit"
                icon={<CheckOutlined />}
                type="primary"
              />
            </Form.Item>
            <Form.Item noStyle>
              <Button icon={<CloseOutlined />} onClick={() => setIsUsernameEditable(false)} />
            </Form.Item>
          </Form>
        )}
      </div>
      <Divider />
      <div className={css.row}>
        <label>Display Name</label>
        {!isDisplaynameEditable ? (
          <div className={css.displayInfo}>
            <span data-testid="text-displayname">
              {currentUser?.displayName || <Typography.Text disabled>N/A</Typography.Text>}
            </span>
            <Button
              data-testid="edit-displayname"
              disabled={!info.userManagementEnabled}
              icon={<EditOutlined />}
              onClick={() => setIsDisplaynameEditable(true)}
            />
          </div>
        ) : (
          <Form
            className={css.form}
            form={displaynameForm}
            layout="inline"
            onFinish={handleSaveDisplayName}>
            <Form.Item initialValue={currentUser?.displayName ?? ''} name="displayName" noStyle>
              <Input maxLength={32} placeholder="Add display name" style={{ widows: '80%' }} />
            </Form.Item>
            <Form.Item noStyle>
              <Button htmlType="submit" icon={<CheckOutlined />} type="primary" />
            </Form.Item>
            <Form.Item noStyle>
              <Button icon={<CloseOutlined />} onClick={() => setIsDisplaynameEditable(false)} />
            </Form.Item>
          </Form>
        )}
      </div>
      {info.userManagementEnabled && (
        <>
          <Divider />
          <div className={css.row}>
            <label>Password</label>
            <Button onClick={PasswordChangeModal.open}>{CHANGE_PASSWORD_TEXT}</Button>
          </div>
          <PasswordChangeModal.Component />
        </>
      )}
    </div>
  );
};

export default SettingsAccount;
