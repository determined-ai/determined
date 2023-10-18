import { Space } from 'antd';
import React, { useCallback, useState } from 'react';

import Drawer from 'components/kit/Drawer';
import Icon from 'components/kit/Icon';
import InlineForm from 'components/kit/InlineForm';
import Input from 'components/kit/Input';
import InputShortcut, { KeyboardShortcut, shortcutToString } from 'components/kit/InputShortcut';
import { useModal } from 'components/kit/Modal';
import Select, { Option } from 'components/kit/Select';
import Spinner from 'components/kit/Spinner';
import useUI, { Mode } from 'components/kit/Theme';
import { makeToast } from 'components/kit/Toast';
import { Loadable } from 'components/kit/utils/loadable';
import PasswordChangeModalComponent from 'components/PasswordChangeModal';
import Section from 'components/Section';
import { ThemeOptions } from 'components/ThemeToggle';
import {
  shortcutSettingsConfig,
  shortcutSettingsDefaults,
  shortcutsSettingsPath,
} from 'components/UserSettings.settings';
import {
  FEATURE_SETTINGS_PATH,
  FEATURES,
  FeatureSettingsConfig,
  ValidFeature,
} from 'hooks/useFeature';
import {
  experimentListGlobalSettingsConfig,
  experimentListGlobalSettingsDefaults,
  experimentListGlobalSettingsPath,
} from 'pages/F_ExpList/F_ExperimentList.settings';
import { TableViewMode } from 'pages/F_ExpList/glide-table/GlideTable';
import { RowHeight, rowHeightItems } from 'pages/F_ExpList/glide-table/OptionsMenu';
import determinedStore from 'stores/determinedInfo';
import userStore from 'stores/users';
import userSettings from 'stores/userSettings';
import handleError, { ErrorType } from 'utils/error';
import { useObservable } from 'utils/observable';

import Accordion from './kit/Accordion';
import Button from './kit/Button';
import Paragraph from './kit/Typography/Paragraph';
import useConfirm from './kit/useConfirm';
import css from './UserSettings.module.scss';
import UserSettingsModalComponent from './UserSettingsModal';

const API_DISPLAYNAME_SUCCESS_MESSAGE = 'Display name updated.';
const API_USERNAME_ERROR_MESSAGE = 'Could not update username.';
const API_USERNAME_SUCCESS_MESSAGE = 'Username updated.';

interface Props {
  show: boolean;
  onClose: () => void;
}

const rowHeightLabels = rowHeightItems.reduce((acc, { rowHeight, label }) => {
  acc[rowHeight] = label;
  return acc;
}, {} as Record<RowHeight, string>);

const UserSettings: React.FC<Props> = ({ show, onClose }: Props) => {
  const currentUser = Loadable.getOrElse(undefined, useObservable(userStore.currentUser));
  const info = useObservable(determinedStore.info);
  const confirm = useConfirm();

  const UserSettingsModal = useModal(UserSettingsModalComponent);
  const PasswordChangeModal = useModal(PasswordChangeModalComponent);
  const {
    ui: { mode: uiMode },
    actions: { setMode },
  } = useUI();

  const currentThemeOption = ThemeOptions[uiMode];

  const handleSaveDisplayName = useCallback(
    async (newValue: string): Promise<void | Error> => {
      try {
        await userStore.patchUser(currentUser?.id || 0, {
          displayName: newValue as string,
        });
        makeToast({ severity: 'Confirm', title: API_DISPLAYNAME_SUCCESS_MESSAGE });
      } catch (e) {
        handleError(e, { silent: false, type: ErrorType.Input });
        return e as Error;
      }
    },
    [currentUser?.id],
  );

  const handleSaveUsername = useCallback(
    async (newValue: string): Promise<void | Error> => {
      try {
        await userStore.patchUser(currentUser?.id || 0, {
          username: newValue as string,
        });
        makeToast({ severity: 'Confirm', title: API_USERNAME_SUCCESS_MESSAGE });
      } catch (e) {
        makeToast({ severity: 'Error', title: API_USERNAME_ERROR_MESSAGE });
        handleError(e, { silent: true, type: ErrorType.Input });
        return e as Error;
      }
    },
    [currentUser?.id],
  );

  const [newPassword, setNewPassword] = useState<string>('');

  const NEW_PASSWORD_REQUIRED_MESSAGE = "Password can't be blank";
  const PASSWORD_TOO_SHORT_MESSAGE = 'Password must have at least 8 characters';
  const PASSWORD_UPPERCASE_MESSAGE = 'Password must include an uppercase letter';
  const PASSWORD_LOWERCASE_MESSAGE = 'Password must include a lowercase letter';
  const PASSWORD_NUMBER_MESSAGE = 'Password must include a number';

  const handleSavePassword = useCallback(
    (value: string) => {
      setNewPassword(value);
      PasswordChangeModal.open();
    },
    [PasswordChangeModal],
  );

  const [editingPassword, setEditingPassword] = useState<boolean>(false);

  return Loadable.match(
    Loadable.all([
      useObservable(
        userSettings.get(experimentListGlobalSettingsConfig, experimentListGlobalSettingsPath),
      ),
      useObservable(userSettings.get(shortcutSettingsConfig, shortcutsSettingsPath)),
      useObservable(userSettings.get(FeatureSettingsConfig, FEATURE_SETTINGS_PATH)),
    ]),
    {
      Failed: () => null,
      Loaded: ([
        savedExperimentListGlobalSettings,
        savedShortcutSettings,
        savedFeatureSettings,
      ]) => {
        const experimentListGlobalSettings = {
          ...experimentListGlobalSettingsDefaults,
          ...(savedExperimentListGlobalSettings ?? {}),
        };
        const shortcutSettings = { ...shortcutSettingsDefaults, ...(savedShortcutSettings ?? {}) };

        return (
          <Drawer open={show} placement="left" title="Settings" onClose={onClose}>
            <Section divider title="Profile">
              <div className={css.section}>
                <InlineForm<string>
                  initialValue={currentUser?.username ?? ''}
                  label="Username"
                  required
                  rules={[{ message: 'Please input your username', required: true }]}
                  testId="username"
                  onSubmit={handleSaveUsername}>
                  <Input autoFocus maxLength={32} placeholder="Add username" />
                </InlineForm>
                <InlineForm<string>
                  initialValue={currentUser?.displayName ?? ''}
                  label="Display Name"
                  testId="displayname"
                  onSubmit={handleSaveDisplayName}>
                  <Input autoFocus maxLength={32} placeholder="Add display name" />
                </InlineForm>
                {info.userManagementEnabled && (
                  <>
                    <InlineForm<string>
                      initialValue={newPassword}
                      isPassword
                      label="Password"
                      open={editingPassword}
                      rules={[
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
                      ]}
                      valueFormatter={(value: string) => {
                        if (value.length) return value;
                        return '*****';
                      }}
                      onCancel={() => setEditingPassword(false)}
                      onEdit={() => setEditingPassword(true)}
                      onSubmit={handleSavePassword}>
                      <Input.Password autoFocus />
                    </InlineForm>
                    <PasswordChangeModal.Component
                      newPassword={newPassword}
                      onSubmit={() => setEditingPassword(false)}
                    />
                  </>
                )}
              </div>
            </Section>
            <Section divider title="Preferences">
              <div className={css.section}>
                <InlineForm<Mode>
                  initialValue={currentThemeOption.className}
                  label="Theme Mode"
                  valueFormatter={(value: Mode) => ThemeOptions[value].displayName}
                  onSubmit={(v) => {
                    setMode(v);
                  }}>
                  <Select searchable={false}>
                    <Option key={ThemeOptions.dark.className} value={ThemeOptions.dark.className}>
                      {ThemeOptions.dark.displayName}
                    </Option>
                    <Option key={ThemeOptions.light.className} value={ThemeOptions.light.className}>
                      {ThemeOptions.light.displayName}
                    </Option>
                    <Option
                      key={ThemeOptions.system.className}
                      value={ThemeOptions.system.className}>
                      {ThemeOptions.system.displayName}
                    </Option>
                  </Select>
                </InlineForm>
                <InlineForm<RowHeight>
                  initialValue={experimentListGlobalSettings.rowHeight}
                  label="Table Density"
                  valueFormatter={(rh) => rowHeightLabels[rh]}
                  onSubmit={(rh) => {
                    userSettings.set(
                      experimentListGlobalSettingsConfig,
                      experimentListGlobalSettingsPath,
                      {
                        ...experimentListGlobalSettings,
                        rowHeight: rh,
                      },
                    );
                  }}>
                  <Select searchable={false}>
                    {rowHeightItems.map(({ rowHeight, label }) => (
                      <Option key={rowHeight} value={rowHeight}>
                        {label}
                      </Option>
                    ))}
                  </Select>
                </InlineForm>
                <InlineForm<TableViewMode>
                  initialValue={experimentListGlobalSettings.tableViewMode}
                  label="Infinite Scroll"
                  valueFormatter={(mode) => (mode === 'scroll' ? 'On' : 'Off')}
                  onSubmit={(mode) => {
                    userSettings.set(
                      experimentListGlobalSettingsConfig,
                      experimentListGlobalSettingsPath,
                      { ...experimentListGlobalSettings, tableViewMode: mode },
                    );
                  }}>
                  <Select searchable={false}>
                    <Option key="scroll" value="scroll">
                      On
                    </Option>
                    <Option key="paged" value="paged">
                      Off
                    </Option>
                  </Select>
                </InlineForm>
              </div>
            </Section>
            <Section divider title="Shortcuts">
              <div className={css.section}>
                <InlineForm<KeyboardShortcut>
                  initialValue={shortcutSettings.omnibar}
                  label="Open Omnibar"
                  valueFormatter={shortcutToString}
                  onSubmit={(sc) => {
                    userSettings.set(shortcutSettingsConfig, shortcutsSettingsPath, {
                      ...shortcutSettings,
                      omnibar: sc,
                    });
                  }}>
                  <InputShortcut />
                </InlineForm>
                <InlineForm<KeyboardShortcut>
                  initialValue={shortcutSettings.jupyterLab}
                  label="Launch JupyterLab Notebook"
                  valueFormatter={shortcutToString}
                  onSubmit={(sc) => {
                    userSettings.set(shortcutSettingsConfig, shortcutsSettingsPath, {
                      ...shortcutSettings,
                      jupyterLab: sc,
                    });
                  }}>
                  <InputShortcut />
                </InlineForm>
                <InlineForm<KeyboardShortcut>
                  initialValue={shortcutSettings.navbarCollapsed}
                  label="Toggle Sidebar"
                  valueFormatter={shortcutToString}
                  onSubmit={(sc) => {
                    userSettings.set(shortcutSettingsConfig, shortcutsSettingsPath, {
                      ...shortcutSettings,
                      navbarCollapsed: sc,
                    });
                  }}>
                  <InputShortcut />
                </InlineForm>
              </div>
            </Section>
            <Section divider title="Experimental">
              <div className={css.section}>
                {Object.entries(FEATURES).map(([feature, description]) => (
                  <InlineForm<boolean>
                    initialValue={
                      savedFeatureSettings?.[feature as ValidFeature] ?? description.defaultValue
                    }
                    key={feature}
                    label={
                      <Space>
                        {description.friendlyName}
                        <Icon name="info" showTooltip title={description.description} />
                      </Space>
                    }
                    valueFormatter={(value) => (value ? 'On' : 'Off')}
                    onSubmit={(val) => {
                      userSettings.set(FeatureSettingsConfig, FEATURE_SETTINGS_PATH, {
                        [feature]: val,
                      });
                    }}>
                    <Select searchable={false}>
                      <Option value={true}>On</Option>
                      <Option value={false}>Off</Option>
                    </Select>
                  </InlineForm>
                ))}
              </div>
            </Section>
            <Section title="Advanced">
              <Paragraph>
                Advanced features are potentially dangerous and could require you to completely
                reset your user settings if you make a mistake.
              </Paragraph>
              <Accordion title="I know what I'm doing">
                <Space>
                  <Button
                    danger
                    type="primary"
                    onClick={() =>
                      confirm({
                        content:
                          'Are you sure you want to reset all user settings to their default values?',
                        onConfirm: () => {
                          setMode(Mode.System);
                          userSettings.clear();
                        },
                        onError: handleError,
                        title: 'Reset User Settings',
                      })
                    }>
                    Reset to Default
                  </Button>
                  <Button onClick={() => UserSettingsModal.open()}>Edit Raw Settings (JSON)</Button>
                  <UserSettingsModal.Component />
                </Space>
              </Accordion>
            </Section>
          </Drawer>
        );
      },
      NotLoaded: () => <Spinner spinning />, // TDOD correctly handle error state
    },
  );
};

export default UserSettings;
