FROM scratch

# Define arguments for building multi-arch images
ARG TARGETPLATFORM

# Copy artifacts to the container
COPY bin/${TARGETPLATFORM}/* /

# Expose ports
EXPOSE 80

# Provide default environment variables
ENV GIN_MODE="release"

# Specify entrypoint and default parameters
ENTRYPOINT ["/ratus"]
