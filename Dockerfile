FROM wav2letter/wav2letter:inference-latest

RUN mkdir /home/model && \
    cd /home/model && \
    for f in acoustic_model.bin tds_streaming.arch decoder_options.json feature_extractor.bin language_model.bin lexicon.txt tokens.txt ; do wget http://dl.fbaipublicfiles.com/wav2letter/inference/examples/model/${f} ; done