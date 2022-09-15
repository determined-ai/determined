import React, { useState } from 'react';
import { useRecoilState } from 'recoil';

import {
  DomainName,
  ProjectDetail,
  userSettingsDomainState,
} from 'recoil/userSettings';
import { updateUserWebSetting } from 'services/api';

const Test: React.FC = () => {
  const [count, setCount] = useState<number>(0);
  const [userWebSettings, setUserWebSettings] = useRecoilState<ProjectDetail>(
    userSettingsDomainState(DomainName.ProjectDetail)
  );
  const onClick = () => {
    setUserWebSettings({
      ...userWebSettings,
      archived: true,
      columnWidths: [],
    });
    updateUserWebSetting({ setting: { value: { new: { f: { count } } } } });
    setCount((prev) => prev + 1);
  };
  return (
    <>
      <button onClick={onClick}>button {count}</button>
      <div>{JSON.stringify(userWebSettings)}</div>
    </>
  );
};

export default Test;
