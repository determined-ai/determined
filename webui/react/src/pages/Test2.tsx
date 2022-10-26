import React, { useState } from 'react';
import { useRecoilState } from 'recoil';

import { config } from 'useWebSettings/useWebSettings.settings';

const Test2: React.FC = () => {
  const [count, setCount] = useState<number>(0);
  const [numOfCake, setNumOfCake] = useRecoilState(config.settings.numOfCake.atom);
  const [letter, setLetter] = useRecoilState(config.settings.letter.atom);

  const onClick = () => {
    setNumOfCake((prev) => prev + 1);
    setLetter(`${numOfCake}`);
    setCount((prev) => prev + 1);
  };

  return (
    <>
      <button onClick={onClick}>button {count}</button>
      <div>{numOfCake}</div>
      <div>{letter}</div>
    </>
  );
};

export default Test2;
