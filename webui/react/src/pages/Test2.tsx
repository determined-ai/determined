import React from 'react';
import { useRecoilState } from 'recoil';

import { DomainName, userSettingsDomainState } from 'recoil/userSettings';

const Test2: React.FC = () => {
  const [ userWebSettings, setUserWebSettings ] = useRecoilState(
    userSettingsDomainState(DomainName.ProjectDetail),
  );
  const onClick = () => setUserWebSettings({ archived: true, columnWidths: [ 12 ] });
  return (
    <>
      <button onClick={onClick}>button</button>
      <div>{JSON.stringify(userWebSettings)}</div>
    </>
  );
};

export default Test2;
