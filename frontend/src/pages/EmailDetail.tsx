import React from 'react';
import { useParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Grid,
  CircularProgress,
  Button,
  Chip,
  Divider,
  Paper,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
} from '@mui/material';
import {
  Summarize as SummarizeIcon,
  Analytics as AnalyticsIcon,
  Person as PersonIcon,
  LocationOn as LocationIcon,
  Business as BusinessIcon,
  Event as EventIcon,
  AttachMoney as MoneyIcon,
} from '@mui/icons-material';
import { api, Email, NEREntity } from '../api/client';

const EntityList: React.FC<{ entities: NEREntity[] }> = ({ entities }) => {
  const getEntityIcon = (type: string) => {
    switch (type.toLowerCase()) {
      case 'person':
        return <PersonIcon />;
      case 'location':
        return <LocationIcon />;
      case 'organization':
        return <BusinessIcon />;
      case 'date':
        return <EventIcon />;
      case 'money':
        return <MoneyIcon />;
      default:
        return <AnalyticsIcon />;
    }
  };

  return (
    <List>
      {entities.map((entity, index) => (
        <ListItem key={index}>
          <ListItemIcon>{getEntityIcon(entity.type)}</ListItemIcon>
          <ListItemText
            primary={entity.text}
            secondary={<Chip label={entity.type} size="small" color="primary" variant="outlined" />}
          />
        </ListItem>
      ))}
    </List>
  );
};

const EmailDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();

  const { data: email, isLoading } = useQuery({
    queryKey: ['email', id],
    queryFn: () => api.getEmail(id!),
    enabled: !!id,
  });

  const summarizeMutation = useMutation({
    mutationFn: () => api.summarizeEmail(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['email', id] });
    },
  });

  const analyzeMutation = useMutation({
    mutationFn: () => api.performNER(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['email', id] });
    },
  });

  if (isLoading || !email) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 3 }}>
        <Typography variant="h4" component="h1">
          {email.subject}
        </Typography>
        <Box sx={{ display: 'flex', gap: 2 }}>
          {!email.summary && (
            <Button
              variant="outlined"
              startIcon={<SummarizeIcon />}
              onClick={() => summarizeMutation.mutate()}
              disabled={summarizeMutation.isPending}
            >
              Summarize
            </Button>
          )}
          {!email.entities && (
            <Button
              variant="outlined"
              startIcon={<AnalyticsIcon />}
              onClick={() => analyzeMutation.mutate()}
              disabled={analyzeMutation.isPending}
            >
              Analyze
            </Button>
          )}
        </Box>
      </Box>

      <Grid container spacing={3}>
        <Grid item xs={12} md={8}>
          <Card>
            <CardContent>
              <Box sx={{ mb: 3 }}>
                <Typography variant="subtitle2" color="text.secondary">
                  From
                </Typography>
                <Typography>{email.from}</Typography>
              </Box>
              <Box sx={{ mb: 3 }}>
                <Typography variant="subtitle2" color="text.secondary">
                  To
                </Typography>
                <Typography>
                  {email.to.map((recipient, index) => (
                    <span key={index}>
                      {recipient}
                      <br />
                    </span>
                  ))}
                </Typography>
              </Box>
              <Box sx={{ mb: 3 }}>
                <Typography variant="subtitle2" color="text.secondary">
                  Received
                </Typography>
                <Typography>{new Date(email.date).toLocaleString()}</Typography>
              </Box>
              <Divider sx={{ my: 2 }} />
              <Box sx={{ whiteSpace: 'pre-wrap' }}>{email.body}</Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={4}>
          {email.summary && (
            <Card sx={{ mb: 3 }}>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Summary
                </Typography>
                <Typography>{email.summary}</Typography>
              </CardContent>
            </Card>
          )}

          {email.entities && email.entities.length > 0 && (
            <Card>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Entities
                </Typography>
                <EntityList entities={email.entities} />
              </CardContent>
            </Card>
          )}
        </Grid>
      </Grid>
    </Box>
  );
};

export default EmailDetail;
