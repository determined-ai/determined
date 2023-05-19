import { Conjunction } from 'components/FilterForm/components/type';
import Select, { Option, SelectValue } from 'components/kit/Select';

import css from './ConjunctionContainer.module.scss';

interface Props {
  index: number;
  conjunction: Conjunction;
  onClick: (value: SelectValue) => void;
}

const ConjunctionContainer = ({ index, conjunction, onClick }: Props): JSX.Element => {
  return (
    <>
      {index === 0 && <div className={css.operator}>Where</div>}
      {index === 1 && (
        <Select searchable={false} value={conjunction} width={'100%'} onChange={onClick}>
          {Object.values(Conjunction).map((c) => (
            <Option key={c} value={c}>
              {c}
            </Option>
          ))}
        </Select>
      )}
      {index > 1 && <div className={css.operator}>{conjunction}</div>}
    </>
  );
};

export default ConjunctionContainer;
