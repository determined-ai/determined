import React, { useState } from 'react';

import useWebSettings, { ProjectDetailKey } from 'recoil/userSettings/useWebSettings';

const Test: React.FC = () => {
  const [ count, setCount ] = useState<number>(0);
  const [ pinned, setPinned ] = useWebSettings(ProjectDetailKey.Pinned);

  const onClick = () => {
    setPinned({ pinned: { 1: [ 1, 2, count + 1 ] } });
    setCount((prev) => prev + 1);
  };
  return (
    <>
      <button onClick={onClick}>button {count}</button>
      <div>{JSON.stringify(pinned)}</div>
    </>
  );
};

export default Test;
