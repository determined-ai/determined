import { Map } from 'immutable';
import { useMemoizedObservable } from 'micro-observables';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { Modal } from 'components/kit/Modal';
import userSettings from 'stores/userSettings';
import { Json } from 'types';
import { isObject } from 'utils/data';
import handleError from 'utils/error';
import { Loadable, Loaded } from 'utils/loadable';

import CodeEditor from './kit/CodeEditor';

interface Props {
  onSave?: () => void;
}

const UserSettingsModal: React.FC<Props> = ({ onSave }: Props) => {
  const initialSettingsString = useMemoizedObservable<Loadable<string>>(
    () =>
      userSettings
        .getAll()
        .select((loadableState) =>
          Loadable.map(loadableState, (state) => JSON.stringify(state, undefined, ' ')),
        ),
    [],
  );
  const [editedSettingsString, setEditedSettingsString] =
    useState<Loadable<string>>(initialSettingsString);

  const editedSettings = useMemo(
    () =>
      Loadable.match(editedSettingsString, {
        Loaded: (settingsString) => {
          try {
            const obj = JSON.parse(settingsString);
            return isObject(obj) ? Map<string, Json>(obj) : undefined;
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

    setEditedSettingsString(initialSettingsString);
  }, [editedSettingsString, initialSettingsString]);

  const handleSave = useCallback(async () => {
    if (!editedSettings) return;

    onSave?.();
    await userSettings.overwrite(editedSettings);
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
