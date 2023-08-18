#!/bin/bash

python run_ner.py \
    --model_name_or_path bert-base-uncased \
    --dataset_name conll2003 \
    --output_dir /tmp/test-ner \
    --max_train_samples 50 \
    --max_eval_samples 50 \
    --max_predict_samples 50 \
    --do_train \
    --do_eval \
    --overwrite_output_dir
