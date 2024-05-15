import Select, { Option, SelectValue } from 'hew/Select';

import { Conjunction } from 'components/FilterForm/components/type';

import css from './ConjunctionContainer.module.scss';

interface Props {
  index: number;
  conjunction: Conjunction;
  onClick: (value: SelectValue) => void;
}

const ConjunctionContainer = ({ index, conjunction, onClick }: Props): JSX.Element => {
  return (
    <>
      {index === 0 && (
        <div className={css.operator} data-test="where">
          Where
        </div>
      )}
      {index === 1 && (
        <Select
          data-test="conjunction"
          searchable={false}
          value={conjunction}
          width={'100%'}
          onChange={onClick}>
          {Object.values(Conjunction).map((c) => (
            <Option key={c} value={c}>
              {c}
            </Option>
          ))}
        </Select>
      )}
      {index > 1 && (
        <div className={css.operator} data-test="conjunctionContinued">
          {conjunction}
        </div>
      )}
    </>
  );
};

export default ConjunctionContainer;
