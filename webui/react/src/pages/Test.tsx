import React, { useState } from 'react';
import { useRecoilState } from 'recoil';

import {
  AllData,
  ProjectDetailKey,
  userSettingsDomainState,
} from 'recoil/userSettings';

const Test: React.FC = () => {
  const [ count, setCount ] = useState<number>(0);
  const [ userWebSettings, setUserWebSettings ] =
    useRecoilState<AllData[ProjectDetailKey.ColumnWidths]>(
      userSettingsDomainState(ProjectDetailKey.ColumnWidths),
    );
  const onClick = () => {
    setUserWebSettings({ columnWidths: [ count + 1 ] });
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
