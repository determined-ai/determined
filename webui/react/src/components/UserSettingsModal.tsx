import { useObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { Modal } from 'components/kit/Modal';
import userSettings from 'stores/userSettings';
import { isObject } from 'utils/data';
import handleError from 'utils/error';
import { Loadable, Loaded } from 'utils/loadable';

import CodeEditor from './kit/CodeEditor';

interface Props {
  onSave?: () => void;
}

const UserSettingsModal: React.FC<Props> = ({ onSave }: Props) => {
  const loadableState = useObservable(userSettings.getAll());
  const stringifiedState = Loadable.map(loadableState, (state) =>
    JSON.stringify(state, undefined, ' '),
  );
  const [editedSettingsString, setEditedSettingsString] =
    useState<Loadable<string>>(stringifiedState);

  const editedSettings: object | undefined = useMemo(
    () =>
      Loadable.match(editedSettingsString, {
        Loaded: (settingsString) => {
          try {
            const obj = JSON.parse(settingsString);
            return isObject(obj) ? obj : undefined;
          } catch {
            return;
          }
        },
        NotLoaded: () => undefined,
      }),
    [editedSettingsString],
  );

  useEffect(() => {
    if (Loadable.isLoaded(editedSettingsString)) return;

    setEditedSettingsString(stringifiedState);
  }, [editedSettingsString, stringifiedState]);

  const handleSave = useCallback(async () => {
    if (!editedSettings) return;

    userSettings.overwrite(editedSettings);
    await onSave?.();
  }, [editedSettings, onSave]);

  const handleChange = useCallback((newSettings: string) => {
    setEditedSettingsString(Loaded(newSettings));
  }, []);

  return (
    <Modal
      cancel
      size="medium"
      submit={{
        disabled: editedSettings === undefined,
        handleError,
        handler: handleSave,
        text: 'Save Settings',
      }}
      title="Edit Raw Settings">
      <CodeEditor
        files={[
          {
            content: editedSettingsString,
            key: 'settings.json',
            title: 'settings.json',
          },
        ]}
        height="400px"
        onChange={handleChange}
        onError={handleError}
      />
    </Modal>
  );
};

export default UserSettingsModal;
