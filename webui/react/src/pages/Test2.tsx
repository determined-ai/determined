import React from 'react';

import useWebSettings, { UserWebSettingsDomain } from 'recoil/userSettings/useWebSettings';

const Test2: React.FC = () => {
  const [projectDetail, setProjectDetail] = useWebSettings(
    UserWebSettingsDomain.ProjectDetail,
    'each',
  );
  const [global, setGlobal] = useWebSettings(UserWebSettingsDomain.Global, 'theme');
  const onClick = () => {
    setGlobal('dark');
    setProjectDetail({ ...projectDetail });
  };
  return (
    <>
      <button onClick={onClick}>button</button>
      <div>{JSON.stringify(projectDetail)}</div>
      <div>{JSON.stringify(global)}</div>
    </>
  );
};

export default Test2;
