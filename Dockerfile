FROM scratch

MAINTAINER Alex Peters <info@alexanderpeters.de>

COPY grafain /

ENTRYPOINT ["/grafain"]
CMD [""]
