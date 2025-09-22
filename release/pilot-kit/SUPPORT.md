# OCX Protocol Support

## Support Channels

### Primary Support
- **Email**: support@ocx-protocol.com
- **Response Time**: 1 business day
- **Hours**: Monday-Friday, 9 AM - 5 PM PST

### Emergency Support
- **Email**: emergency@ocx-protocol.com
- **Response Time**: 15 minutes
- **Hours**: 24/7 for critical issues
- **Requirements**: P0 severity issues only

### Community Support
- **GitHub Issues**: https://github.com/ocx-protocol/ocx/issues
- **Discord**: https://discord.gg/ocx-protocol
- **Documentation**: https://docs.ocx-protocol.com

## Support Tiers

### Pilot Support (Free)
- **Response Time**: 2 business days
- **Channels**: Email, GitHub Issues
- **Scope**: Installation, basic configuration, bug reports
- **Duration**: 2 weeks pilot period

### Professional Support ($299/month)
- **Response Time**: 1 business day
- **Channels**: Email, GitHub Issues, Discord
- **Scope**: All pilot features + production support, performance tuning
- **Includes**: 10M verifications/month, email support

### Enterprise Support (Custom)
- **Response Time**: 4 hours
- **Channels**: Email, GitHub Issues, Discord, Phone, Slack
- **Scope**: All professional features + dedicated support, SLA guarantees
- **Includes**: Unlimited verifications, dedicated support channel, 1-hour incident response

## Response Times

### Severity Levels

#### P0 (Critical) - 15 minutes
- Service completely down
- Security breach
- Data loss or corruption
- Complete verification failure

#### P1 (High) - 1 hour
- Service degraded performance
- High error rates
- Security vulnerability
- Key rotation issues

#### P2 (Medium) - 4 hours
- Feature not working as expected
- Performance issues
- Configuration problems
- Integration issues

#### P3 (Low) - 24 hours
- Feature requests
- Documentation questions
- General inquiries
- Enhancement requests

### Escalation Procedures

#### Level 1: Initial Response
- **P0**: 15 minutes
- **P1**: 1 hour
- **P2**: 4 hours
- **P3**: 24 hours

#### Level 2: Escalation
- **P0**: 30 minutes if no response
- **P1**: 2 hours if no response
- **P2**: 8 hours if no response
- **P3**: 48 hours if no response

#### Level 3: Management
- **P0**: 1 hour if no response
- **P1**: 4 hours if no response
- **P2**: 24 hours if no response
- **P3**: 72 hours if no response

## Support Scope

### Included Support
- **Installation**: Help with installation and setup
- **Configuration**: Assistance with configuration
- **Troubleshooting**: Debugging and issue resolution
- **Documentation**: Clarification of documentation
- **Bug Reports**: Investigation and fixing of bugs
- **Performance**: Basic performance tuning
- **Integration**: Basic integration assistance

### Excluded Support
- **Custom Development**: Custom feature development
- **Third-Party Issues**: Issues with third-party software
- **Hardware Issues**: Physical hardware problems
- **Network Issues**: Network infrastructure problems
- **Training**: Comprehensive training programs
- **Consulting**: Strategic consulting services

## Getting Help

### Before Contacting Support

1. **Check Documentation**: Review relevant documentation
2. **Search Issues**: Search GitHub issues for similar problems
3. **Run Diagnostics**: Use provided diagnostic scripts
4. **Gather Information**: Collect relevant logs and information

### Information to Include

#### For Bug Reports
- **Version**: OCX Protocol version
- **Environment**: OS, Docker, Kubernetes versions
- **Steps to Reproduce**: Detailed steps to reproduce the issue
- **Expected Behavior**: What should happen
- **Actual Behavior**: What actually happens
- **Logs**: Relevant log files
- **Configuration**: Relevant configuration files

#### For Performance Issues
- **Metrics**: Current performance metrics
- **Load Test Results**: Results from load tests
- **Resource Usage**: CPU, memory, disk usage
- **Database Performance**: Database metrics and queries
- **Network**: Network latency and throughput

#### For Integration Issues
- **Client Code**: Relevant client code
- **API Calls**: API requests and responses
- **Error Messages**: Complete error messages
- **Environment**: Development vs production environment
- **Dependencies**: Third-party dependencies

### Diagnostic Scripts

#### Health Check
```bash
# Basic health check
curl -s http://localhost:8080/health | jq

# Comprehensive health check
bash scripts/smoke.sh
```

#### Performance Check
```bash
# Load test
bash scripts/load_test.sh 200 60

# Metrics check
curl -s http://localhost:8080/metrics | grep ocx_
```

#### Database Check
```bash
# Database connectivity
psql $DATABASE_URL -c "SELECT 1;"

# Database performance
psql $DATABASE_URL -c "SELECT * FROM pg_stat_activity;"
```

## Service Level Agreements (SLAs)

### Availability SLA
- **Target**: 99.9% uptime
- **Measurement**: Successful responses to `/health` endpoint
- **Exclusions**: Planned maintenance, external dependencies
- **Credits**: Service credits for SLA violations

### Performance SLA
- **P99 Latency**: < 20ms for verify endpoint
- **Throughput**: 200+ RPS per node
- **Error Rate**: < 0.1% error rate
- **Measurement**: 5-minute rolling average

### Support SLA
- **Response Time**: As per severity levels above
- **Resolution Time**: 24 hours for P0, 72 hours for P1
- **Escalation**: Automatic escalation if not met
- **Credits**: Service credits for SLA violations

## Maintenance Windows

### Planned Maintenance
- **Schedule**: First Sunday of each month, 2 AM - 4 AM PST
- **Notification**: 48 hours advance notice
- **Duration**: Maximum 2 hours
- **Impact**: Brief service interruption

### Emergency Maintenance
- **Schedule**: As needed for critical issues
- **Notification**: Immediate notification
- **Duration**: As short as possible
- **Impact**: Service interruption until resolved

## Security Issues

### Reporting Security Issues
- **Email**: security@ocx-protocol.com
- **Response Time**: 4 hours
- **Scope**: Security vulnerabilities and incidents
- **Confidentiality**: Strict confidentiality maintained

### Security Response Process
1. **Acknowledgment**: Within 4 hours
2. **Assessment**: Within 24 hours
3. **Fix Development**: Within 72 hours
4. **Deployment**: Within 7 days
5. **Disclosure**: Coordinated disclosure

## Training and Resources

### Documentation
- **API Reference**: Complete API documentation
- **Runbooks**: Operational procedures
- **Security Guide**: Security best practices
- **Operations Guide**: Operational procedures

### Training Materials
- **Quick Start Guide**: Getting started quickly
- **Integration Guide**: Integration best practices
- **Troubleshooting Guide**: Common issues and solutions
- **Video Tutorials**: Step-by-step video guides

### Community Resources
- **GitHub Repository**: Source code and issues
- **Discord Community**: Real-time chat support
- **Blog**: Technical articles and updates
- **Webinars**: Regular training webinars

## Contact Information

### Support Team
- **Primary**: support@ocx-protocol.com
- **Emergency**: emergency@ocx-protocol.com
- **Security**: security@ocx-protocol.com
- **Sales**: sales@ocx-protocol.com

### Management
- **Engineering Director**: engineering@ocx-protocol.com
- **CTO**: cto@ocx-protocol.com
- **CEO**: ceo@ocx-protocol.com

### Office Information
- **Address**: 1234 Protocol Street, San Francisco, CA 94105
- **Phone**: +1 (555) 123-4567
- **Hours**: Monday-Friday, 9 AM - 5 PM PST

## Feedback

### Feedback Channels
- **Product Feedback**: product@ocx-protocol.com
- **Feature Requests**: features@ocx-protocol.com
- **Documentation**: docs@ocx-protocol.com
- **General**: feedback@ocx-protocol.com

### Feedback Process
1. **Submission**: Submit feedback via email or GitHub
2. **Review**: Team reviews feedback within 1 week
3. **Response**: Response provided within 2 weeks
4. **Implementation**: Considered for future releases
5. **Follow-up**: Updates on implementation status
