import React from 'react';
import { useRecoilState } from 'recoil';

import { AllData, ProjectDetailKey, userSettingsDomainState } from 'recoil/userSettings';

const Test2: React.FC = () => {
  const [ userWebSettings, setUserWebSettings ] =
   useRecoilState<AllData[ProjectDetailKey.Archived]>(
     userSettingsDomainState(ProjectDetailKey.Archived),
   );
  const onClick = () => setUserWebSettings({ archived: false });
  return (
    <>
      <button onClick={onClick}>button</button>
      <div>{JSON.stringify(userWebSettings)}</div>
    </>
  );
};

export default Test2;
