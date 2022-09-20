import React, { useState } from 'react';

import useWebSettings, { UserWebSettingsDomain } from 'recoil/userSettings/useWebSettings';

const Test: React.FC = () => {
  const [count, setCount] = useState<number>(0);
  const [projectDetail, setProjectDetail] = useWebSettings(
    UserWebSettingsDomain.ProjectDetail,
    'each',
  );

  const onClick = () => {
    setProjectDetail({ ...projectDetail });
    setCount((prev) => prev + 1);
  };
  return (
    <>
      <button onClick={onClick}>button {count}</button>
      <div>{JSON.stringify(projectDetail)}</div>
    </>
  );
};

export default Test;
