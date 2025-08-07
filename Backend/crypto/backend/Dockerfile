FROM golang:1.22.2

# Set working directory
WORKDIR app

# Copy go files
COPY . .

# Build the app
RUN go build -o app crypto.go

# Expose port
EXPOSE 8000

# Run it
CMD ["./app"]
