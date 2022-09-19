import { Tooltip, Typography } from 'antd';
import React, { useMemo } from 'react';

import HumanReadableNumber from 'components/HumanReadableNumber';
import Section from 'components/Section';
import { unflattenObject } from 'shared/utils/data';
import { clamp } from 'shared/utils/number';
import {
  ExperimentBase, HyperparameterType, TrialDetails,
} from 'types';

import css from './TrialRangeHyperparameters.module.scss';

export interface Props {
  experiment: ExperimentBase;
  trial: TrialDetails;
}

interface HyperParameter {
  name: string;
  range: [number, number];
  type: HyperparameterType;
  val: string;
  vals: string[];
}

const TrialRangeHyperparameters: React.FC<Props> = ({ experiment, trial }: Props) => {
  const hyperparameters: HyperParameter[] = useMemo(() => {
    return Object.entries(experiment.hyperparameters).map(([ name, value ]) => {
      return {
        name: name,
        range: value.type === HyperparameterType.Log ?
          [ (value.base ?? 10) ** (value.minval ?? -5),
            (value.base ?? 10) ** (value.maxval ?? 1) ] :
          [ value.minval ?? 0, value.maxval ?? 1 ],
        type: value.type,
        val: JSON.stringify(trial.hyperparameters[name] ??
          unflattenObject(trial.hyperparameters)[name] ?? 0),
        vals: value.vals?.map((val) => JSON.stringify(val)) ??
          [ JSON.stringify(value.minval ?? 0), JSON.stringify(value.maxval ?? 1) ],
      };
    });
  }, [ experiment.hyperparameters, trial.hyperparameters ]);

  return (
    <div className={css.base}>
      <Section bodyBorder bodyScroll>
        <div className={css.container}>
          {hyperparameters.map((hp) => (
            <div key={hp.name}>
              <HyperparameterRange hp={hp} />
            </div>
          ))}
        </div>
      </Section>
    </div>
  );
};

interface RangeProps {
  hp: HyperParameter
}

const HyperparameterRange:React.FC<RangeProps> = ({ hp }: RangeProps) => {
  const pointerPosition = useMemo(() => {
    switch (hp.type) {
      case HyperparameterType.Constant:
        return 0.5;
      case HyperparameterType.Categorical:
      {
        const idx = hp.vals.indexOf(hp.val);
        return ((idx === -1 ? 0 : idx) / (hp.vals.length - 1));
      }
      case HyperparameterType.Log:
        return clamp(1 - Math.log(parseFloat(hp.val) / hp.range[0]) /
            (Math.log(hp.range[1] / hp.range[0])), 0, 1);
      default:
        return clamp(1 - (parseFloat(hp.val) - hp.range[0]) /
            (hp.range[1] - hp.range[0]), 0, 1);
    }
  }, [ hp ]);

  return (
    <div className={css.hpContainer}>
      <p className={css.title}>{hp.name}</p>
      <div className={css.innerContainer}>
        <ValuesTrack hp={hp} />
        <MainTrack hp={hp} />
        <div className={css.pointerTrack}>
          <Pointer
            containerStyle={{ transform: `translateY(${270 * pointerPosition}px)` }}
            content={<ParsedHumanReadableValue hp={hp} />}
          />
        </div>
      </div>
    </div>
  );
};

interface TrackProps {
  hp: HyperParameter
}

const ValuesTrack: React.FC<TrackProps> = ({ hp }: TrackProps) => {
  switch (hp.type) {
    case HyperparameterType.Constant:
      return <div className={css.valuesTrack} />;
    case HyperparameterType.Categorical:
      return (
        <div className={css.valuesTrack}>
          {hp.vals.map((option) => (
            <Typography.Paragraph
              ellipsis={{ rows: 1, tooltip: true }}
              key={option.toString()}>
              <p className={css.text}>{option}</p>
            </Typography.Paragraph>
          ))}
        </div>
      );
    case HyperparameterType.Log:
      return (
        <div className={css.valuesTrack}>
          {(new Array(Math.floor(Math.log10((hp.range[1]) / (hp.range[0])) + 1))).fill(null)
            .map((_, idx) => (
              <p className={css.text} key={idx}>
                {JSON.stringify((hp.range[1]) / (10 ** idx)).length > 4 ?
                  ((hp.range[1]) / (10 ** idx)).toExponential() :
                  (hp.range[1]) / (10 ** idx)}
              </p>
            ))}
        </div>
      );
    default:
      return (
        <div className={css.valuesTrack}>
          <p className={css.text}>{hp.range[1]}</p>
          <p className={css.text}>{hp.range[0]}</p>
        </div>
      );
  }
};

const MainTrack: React.FC<TrackProps> = ({ hp }: TrackProps) => {
  let trackType;
  let content;
  switch (hp.type) {
    case HyperparameterType.Categorical:
      trackType = css.grayTrack;
      content = hp.vals.map((option) => (
        <div className={css.trackOption} key={option.toString()} />
      ));
      break;
    case HyperparameterType.Constant:
      trackType = css.constantTrack;
      content = <div className={css.trackOption} />;
      break;
    case HyperparameterType.Log:
      trackType = css.blueTrack;
      content = (new Array(Math.floor(Math.log10((hp.range[1]) / (hp.range[0])) + 1)))
        .fill(null)
        .map((_, idx) => <div className={css.tick} key={idx} />);
      break;
    default:
      trackType = css.blueTrack;
      content = hp.vals.map((option) => (
        <div className={css.trackOption} key={option.toString()} />
      ));
  }
  return (
    <div className={trackType}>
      {content}
    </div>
  );
};

interface PHRVProps {
  hp: HyperParameter
}

const ParsedHumanReadableValue: React.FC<PHRVProps> = ({ hp }: PHRVProps) => {
  switch (hp.type) {
    case HyperparameterType.Categorical:
    case HyperparameterType.Constant:
      return (
        <Typography.Paragraph ellipsis={{ rows: 1, tooltip: true }}>
          <p className={css.text}>{hp.val}</p>
        </Typography.Paragraph>
      );
    case HyperparameterType.Double:
      return (
        <p className={css.text}>
          <HumanReadableNumber num={parseFloat(hp.val as string)} precision={3} />
        </p>
      );
    case HyperparameterType.Int:
      return <p className={css.text}>{parseInt(hp.val as string)}</p>;
    case HyperparameterType.Log:
      return (
        <Tooltip title={hp.val}>
          <p className={css.text}>{parseFloat(hp.val as string).toExponential(2)}</p>
        </Tooltip>
      );
    default:
      return <p className={css.text}>{hp.val}</p>;
  }
};

interface PointerProps {
  containerStyle: React.CSSProperties;
  content: JSX.Element;
}

const Pointer: React.FC<PointerProps> = ({ containerStyle, content }: PointerProps) => {
  return (
    <div className={css.pointerContainer} style={containerStyle}>
      <div className={css.pointerArrow} />
      <div className={css.pointerText}>{content}</div>
    </div>
  );
};

export default TrialRangeHyperparameters;
